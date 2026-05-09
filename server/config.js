const path = require('path');

module.exports = {
  port: parseInt(process.env.PORT || '3000', 10),
  qosKey: process.env.QOS_KEY || '',
  dataDir: path.resolve(__dirname, '..', 'data'),
  qosHttpBase: 'https://api.qos.hk',
  qosWsUrl: `wss://api.qos.hk/ws?key=${process.env.QOS_KEY || ''}`,
};
