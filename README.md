go-gripcontrol
================

Author: Konstantin Bokarius <kon@fanout.io>

A GRIP library for Go.

License
-------

go-gripcontrol is offered under the MIT license. See the LICENSE file.

Installation
------------

```sh
```

Usage
-----

Examples for how to publish HTTP response and HTTP stream messages to GRIP proxy endpoints via the GripPubControl class.

```go
package main

import "github.com/fanout/go-pubcontrol"
import "github.com/fanout/go-gripcontrol"
import "encoding/base64"

func main() {
    // GripPubControl can be initialized with or without an endpoint configuration.
    // Each endpoint can include optional JWT authentication info.
    // Multiple endpoints can be included in a single configuration.

    // Initialize GripPubControl with a single endpoint:
    decodedKey, err := base64.StdEncoding.DecodeString("<key>")
    if err != nil {
        panic("Failed to base64 decode the key")
    }
    pub := gripcontrol.NewGripPubControl([]map[string]interface{} {
            map[string]interface{} {
            "control_uri": "https://api.fanout.io/realm/<realm>",
            "control_iss": "<realm>", 
            "key": decodedKey}})

    // Add new endpoints by applying an endpoint configuration:
    pub.ApplyGripConfig([]map[string]interface{} {
            map[string]interface{} { "control_uri": "<myendpoint_uri_1>" },
            map[string]interface{} { "control_uri": "<myendpoint_uri_2>" }})

    // Remove all configured endpoints:
    pub.RemoveAllClients()

    // Explicitly add an endpoint as a PubControlClient instance:
    client := pubcontrol.NewPubControlClient("<myendpoint_uri>")
    // Optionally set JWT auth: client.SetAuthJwt(<claim>, "<key>")
    // Optionally set basic auth: client.SetAuthBasic("<user>", "<password>")
    pub.AddClient(client)

    // Publish across all configured endpoints:
    err = pub.PublishHttpResponse("<channel>", "Test publish!!", "", "")
    if err != nil {
        panic("Publish failed with: " + err.Error())
    }
    err = pub.PublishHttpStream("<channel>", "Test publish!!", "", "")
    if err != nil {
        panic("Publish failed with: " + err.Error())
    }
}
```

Validate the Grip-Sig request header from incoming GRIP messages. This ensures that the message was sent from a valid source and is not expired. Note that when using Fanout.io the key is the realm key, and when using Pushpin the key is configurable in Pushpin's settings.

```go
isValid := gripcontrol.ValidateSig(request.Header["Grip-Sig"][0], "<key>")
```

Long polling example via response _headers_. The client connects to a GRIP proxy over HTTP and the proxy forwards the request to the origin. The origin subscribes the client to a channel and instructs it to long poll via the response _headers_. Note that with the recent versions of Apache it's not possible to send a 304 response containing custom headers, in which case the response body should be used instead (next usage example below).

```go
package main

import "github.com/fanout/go-gripcontrol"
import "net/http"

func HandleRequest(writer http.ResponseWriter, request *http.Request) {
    // Validate the Grip-Sig header:
    if !gripcontrol.ValidateSig(request.Header["Grip-Sig"][0], "<key>") {
        http.Error(writer, "GRIP authorization failed", http.StatusUnauthorized)
        return
    }

    // Create channel header containing channel information:
    channel := gripcontrol.CreateGripChannelHeader([]*gripcontrol.Channel {
            &gripcontrol.Channel{Name: "<channel>"}})

    // Instruct the client to long poll via the response headers:
    writer.Header().Set("Grip-Hold", "response")
    writer.Header().Set("Grip-Channel", channel)
    // To optionally set a timeout value in seconds:
    // writer.Header().Set("Grip-Timeout", "<timeout_value>")
}

func main() {
    http.HandleFunc("/", HandleRequest)
    http.ListenAndServe(":80", nil)
}
```

Long polling example via response _body_. The client connects to a GRIP proxy over HTTP and the proxy forwards the request to the origin. The origin subscribes the client to a channel and instructs it to long poll via the response _body_.

```go
package main

import "github.com/fanout/go-gripcontrol"
import "net/http"
import "io"

func HandleRequest(writer http.ResponseWriter, request *http.Request) {
    // Validate the Grip-Sig header:
    if !gripcontrol.ValidateSig(request.Header["Grip-Sig"][0], "<key>") {
        http.Error(writer, "GRIP authorization failed", http.StatusUnauthorized)
        return
    }

    // Create channel list containing channel information:
    channel := []*gripcontrol.Channel {&gripcontrol.Channel{Name: "<channel>"}}

    // Create hold response body:
    body, err := gripcontrol.CreateHoldResponse(channel, nil, nil)
    // Or to optionally set a timeout value in seconds:
    // timeout := <timeout_value>
    // body, err := gripcontrol.CreateHoldResponse(channel, nil, &timeout)
    if err != nil {
        panic("Failed to create hold response: " + err.Error())
    }

    // Instruct the client to long poll via the response body:
    writer.Header().Set("Content-Type", "application/grip-instruct")
    io.WriteString(writer, body)
}

func main() {
    http.HandleFunc("/", HandleRequest)
    http.ListenAndServe(":80", nil)
}
```

WebSocket example using golang.org/x/net/websocket. A client connects to a GRIP proxy via WebSockets and the proxy forward the request to the origin. The origin accepts the connection over a WebSocket and responds with a control message indicating that the client should be subscribed to a channel. Note that in order for the GRIP proxy to properly interpret the control messages, the origin must provide a 'grip' extension in the 'Sec-WebSocket-Extensions' header.

```go
package main

import "time"
import "net/http"
import "github.com/gorilla/websocket"
import "github.com/fanout/go-pubcontrol"
import "github.com/fanout/go-gripcontrol"

var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool { return true },
}

func GripWebSocketHandler(writer http.ResponseWriter, request *http.Request) {
    // Create the WebSocket control message:
    wsControlMessage, err := gripcontrol.WebSocketControlMessage("subscribe",
            map[string]interface{} { "channel": "<channel>" })
    if err != nil {
        panic("Unable to create control message: " + err.Error())
    }

    // Ensure that the GRIP proxy processes control messages by upgrading
    // with the Sec-WebSocket-Extensions header:
    conn, _ := upgrader.Upgrade(writer, request, http.Header {
            "Sec-WebSocket-Extensions": []string {"grip; message-prefix=\"\""}})

    // Subscribe the WebSocket to a channel:
    conn.WriteMessage(1, []byte("c:" + wsControlMessage))

    // Wait 3 seconds and publish a message to the subscribed channel:
    time.Sleep(3 * time.Second)
    pub := gripcontrol.NewGripPubControl([]map[string]interface{} {
            map[string]interface{} { "control_uri": "<myendpoint_uri>" }})
    format := &gripcontrol.WebSocketMessageFormat {
            Content: []byte("Test WebSocket Publish!!") } 
    item := pubcontrol.NewItem([]pubcontrol.Formatter{format}, "", "")
    err = pub.Publish("test_channel", item)
    if err != nil {
        panic("Publish failed with: " + err.Error())
    }
}

func main() {
    http.HandleFunc("/", GripWebSocketHandler)
    http.ListenAndServe(":80", nil)
}
```

WebSocket over HTTP example using the WEBrick gem. In this case, a client connects to a GRIP proxy via WebSockets and the GRIP proxy communicates with the origin via HTTP.

```go
require 'webrick'
require 'gripcontrol'

class GripWebSocketOverHttpResponse < WEBrick::HTTPServlet::AbstractServlet
  def do_POST(request, response)
    # Validate the Grip-Sig header:
    if !GripControl.validate_sig(request['Grip-Sig'], '<key>')
      return
    end

    # Set the headers required by the GRIP proxy:
    response.status = 200
    response['Sec-WebSocket-Extensions'] = 'grip; message-prefix=""'
    response['Content-Type'] = 'application/websocket-events'

    in_events = GripControl.decode_websocket_events(request.body)
    if in_events[0].type == 'OPEN'
      # Open the WebSocket and subscribe it to a channel:
      out_events = []
      out_events.push(WebSocketEvent.new('OPEN'))
      out_events.push(WebSocketEvent.new('TEXT', 'c:' +
          GripControl.websocket_control_message('subscribe',
          {'channel' => '<channel>'})))
      response.body = GripControl.encode_websocket_events(out_events)
      Thread.new { publish_message }
    end
  end

  def publish_message
    # Wait and then publish a message to the subscribed channel:
    sleep(3)
    grippub = GripPubControl.new({'control_uri' => '<myendpoint>'})
    grippub.publish('<channel>', Item.new(
        WebSocketMessageFormat.new('Test WebSocket publish!!')))
  end
end

server = WEBrick::HTTPServer.new(Port: 80)
server.mount "/websocket", GripWebSocketOverHttpResponse
trap "INT" do server.shutdown end
server.start
```

Parse a GRIP URI to extract the URI, ISS, and key values. The values will be returned in a hash containing 'control_uri', 'control_iss', and 'key' keys.

```go
config = GripControl.parse_grip_uri(
    'http://api.fanout.io/realm/<myrealm>?iss=<myrealm>' +
    '&key=base64:<myrealmkey>')
```
