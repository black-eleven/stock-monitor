package qos

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type KlineRequest struct {
	C  string `json:"c"`
	E  int64  `json:"e,omitempty"`
	Co int    `json:"co"`
	A  int    `json:"a"`
	Kt int    `json:"kt"`
}

func (c *QosClient) FetchHistoryKline(code string, kt int, count int) ([]json.RawMessage, error) {
	if !c.IsConnected() {
		return nil, errors.New("not connected")
	}

	reqid := c.reqSeq.Add(1)
	endTs := time.Now().Unix()

	ch := make(chan rawResponse, 1)
	c.pending.Store(reqid, ch)

	req := struct {
		Type      string         `json:"type"`
		KlineReqs []KlineRequest `json:"kline_reqs"`
		Reqid     int64          `json:"reqid"`
	}{
		Type: "RH",
		KlineReqs: []KlineRequest{{
			C: code, E: endTs, Co: count, A: 0, Kt: kt,
		}},
		Reqid: reqid,
	}

	body, _ := json.Marshal(req)
	if err := c.send(body); err != nil {
		c.pending.Delete(reqid)
		return nil, err
	}

	select {
	case resp := <-ch:
		if resp.err != nil {
			return nil, resp.err
		}
		var msg struct {
			Data json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(resp.msg, &msg); err != nil {
			return nil, err
		}
		var data []json.RawMessage
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			return nil, err
		}
		if data == nil {
			data = []json.RawMessage{}
		}
		return data, nil
	case <-time.After(10 * time.Second):
		c.pending.Delete(reqid)
		return nil, errors.New("history kline request timeout")
	}
}

func (c *QosClient) FetchQuote(code string) (*Quote, error) {
	if !c.IsConnected() {
		return nil, errors.New("not connected")
	}

	reqid := c.reqSeq.Add(1)
	ch := make(chan rawResponse, 1)
	c.pending.Store(reqid, ch)

	req := map[string]interface{}{
		"type":  "RS",
		"codes": []string{code},
		"reqid": reqid,
	}

	body, _ := json.Marshal(req)
	if err := c.send(body); err != nil {
		c.pending.Delete(reqid)
		return nil, err
	}

	select {
	case resp := <-ch:
		if resp.err != nil {
			return nil, resp.err
		}
		var msg struct {
			Data []struct {
				C  string `json:"c"`
				Lp string `json:"lp"`
				Yp string `json:"yp"`
				O  string `json:"o"`
				H  string `json:"h"`
				L  string `json:"l"`
				V  string `json:"v"`
				T  string `json:"t"`
				Ts int64  `json:"ts"`
				S  interface{} `json:"s"`
			} `json:"data"`
		}
		if err := json.Unmarshal(resp.msg, &msg); err != nil {
			return nil, err
		}
		if len(msg.Data) == 0 {
			return nil, fmt.Errorf("no data")
		}
		d := msg.Data[0]
		return &Quote{
			Code: d.C, Price: parseFloat(d.Lp), YP: parseFloat(d.Yp),
			Open: parseFloat(d.O), High: parseFloat(d.H), Low: parseFloat(d.L),
			Volume: parseFloat(d.V), Turnover: parseFloat(d.T),
			Timestamp: d.Ts, Status: fmt.Sprint(d.S),
		}, nil
	case <-time.After(10 * time.Second):
		c.pending.Delete(reqid)
		return nil, fmt.Errorf("quote request timeout for %s", code)
	}
}
