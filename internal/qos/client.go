package qos

import (
	"encoding/json"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type Quote struct {
	Code      string  `json:"code"`
	Price     float64 `json:"price"`
	YP        float64 `json:"yp"`
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Volume    float64 `json:"volume"`
	Turnover  float64 `json:"turnover"`
	Timestamp int64   `json:"timestamp"`
	Status    string  `json:"status"`
}

type Kline struct {
	Code      string  `json:"code"`
	Open      float64 `json:"open"`
	Close     float64 `json:"close"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Volume    float64 `json:"volume"`
	Timestamp int64   `json:"timestamp"`
	Kt        int     `json:"kt"`
}

type rawResponse struct {
	msg []byte
	err error
}

type QosClient struct {
	wsUrl    string
	conn     *websocket.Conn
	mu       sync.Mutex
	connected atomic.Bool

	sendCh  chan []byte
	closeCh chan struct{}

	pending sync.Map   // int64 → chan rawResponse
	reqSeq  atomic.Int64

	OnQuote func(Quote)
	OnKline func(Kline)

	reconnectDelay time.Duration
	heartbeatTimer *time.Timer

	subscribedCodes []string
}

func NewClient(wsUrl string) *QosClient {
	return &QosClient{
		wsUrl:         wsUrl,
		sendCh:        make(chan []byte, 64),
		closeCh:       make(chan struct{}),
		reconnectDelay: 1 * time.Second,
	}
}

func (c *QosClient) Connect() {
	for {
		select {
		case <-c.closeCh:
			return
		default:
		}

		conn, _, err := websocket.DefaultDialer.Dial(c.wsUrl, nil)
		if err != nil {
			log.Printf("[QOS] Connect error: %v (retry in %v)", err, c.reconnectDelay)
			select {
			case <-time.After(c.reconnectDelay):
			case <-c.closeCh:
				return
			}
			c.reconnectDelay *= 2
			if c.reconnectDelay > 30*time.Second {
				c.reconnectDelay = 30 * time.Second
			}
			continue
		}

		log.Println("[QOS] Connected")
		c.mu.Lock()
		c.conn = conn
		c.mu.Unlock()
		c.connected.Store(true)
		c.reconnectDelay = 1 * time.Second

		// Re-subscribe after reconnect
		if len(c.subscribedCodes) > 0 {
			c.subscribe(c.subscribedCodes)
		}

		go c.readLoop(conn)
		c.writeLoop(conn)

		c.connected.Store(false)
		log.Println("[QOS] Disconnected, reconnecting...")
	}
}

func (c *QosClient) readLoop(conn *websocket.Conn) {
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return
		}

		var msg struct {
			Tp    string          `json:"tp"`
			Type  string          `json:"type"`
			Reqid int64           `json:"reqid"`
			Data  json.RawMessage `json:"data"`
			C     string          `json:"c"`
			Lp    string          `json:"lp"`
			Yp    string          `json:"yp"`
			O     string          `json:"o"`
			H     string          `json:"h"`
			L     string          `json:"l"`
			V     string          `json:"v"`
			T     string          `json:"t"`
			Ts    int64           `json:"ts"`
			S     string          `json:"s"`
			Cl    string          `json:"cl"`
			Kt    int             `json:"kt"`
		}

		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}

		switch {
		case msg.Tp == "S" && c.OnQuote != nil:
			c.OnQuote(Quote{
				Code: msg.C, Price: parseFloat(msg.Lp), YP: parseFloat(msg.Yp),
				Open: parseFloat(msg.O), High: parseFloat(msg.H), Low: parseFloat(msg.L),
				Volume: parseFloat(msg.V), Turnover: parseFloat(msg.T),
				Timestamp: msg.Ts, Status: msg.S,
			})

		case msg.Tp == "K" && c.OnKline != nil:
			c.OnKline(Kline{
				Code: msg.C, Open: parseFloat(msg.O), Close: parseFloat(msg.Cl),
				High: parseFloat(msg.H), Low: parseFloat(msg.L),
				Volume: parseFloat(msg.V), Timestamp: msg.Ts, Kt: msg.Kt,
			})

		case msg.Type == "RH" || msg.Type == "RK" || msg.Type == "RS":
			if ch, ok := c.pending.LoadAndDelete(msg.Reqid); ok {
				ch.(chan rawResponse) <- rawResponse{msg: raw}
			}
		}
	}
}

func (c *QosClient) writeLoop(conn *websocket.Conn) {
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case data := <-c.sendCh:
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		case <-heartbeat.C:
			c.send([]byte(`{"type":"H"}`))
		}
	}
}

func (c *QosClient) send(data []byte) {
	select {
	case c.sendCh <- data:
	default:
		log.Println("[QOS] send buffer full, dropping message")
	}
}

func (c *QosClient) Send(data []byte) {
	c.send(data)
}

func (c *QosClient) subscribe(codes []string) {
	c.subscribedCodes = append([]string{}, codes...)
	if c.connected.Load() {
		c.sendJSON(map[string]interface{}{"type": "S", "codes": []string{joinCodes(codes)}})
	}
}

func (c *QosClient) IsConnected() bool {
	return c.connected.Load()
}

func (c *QosClient) Close() {
	close(c.closeCh)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *QosClient) sendJSON(v interface{}) {
	data, _ := json.Marshal(v)
	c.send(data)
}

func joinCodes(codes []string) string {
	s := ""
	for i, c := range codes {
		if i > 0 {
			s += ","
		}
		s += c
	}
	return s
}

func parseFloat(s string) float64 {
	var f float64
	json.Unmarshal([]byte(s), &f)
	return f
}
