import { WebSocketServer } from 'ws';

/**
 * This websocket server is here just to simulate a digitalstrom server and how to reconnect
 * @type {WebSocketServer}
 */

const wss = new WebSocketServer({ port: 8090 });

wss.on('connection', function connection(ws) {
    ws.on('message', function message(data) {
        console.log('received: %s', data);
    });

    // ws.send('something');
});

console.log('Started')