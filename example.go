package main

import (
	"net/url"
	"sync/atomic"

	"github.com/gorilla/websocket"
	"github.com/k0kubun/pp"
	log "github.com/sirupsen/logrus"
)

func doit() {
	u := url.URL{Scheme: "ws", Host: "localhost:1337", Path: "/ws-api"}
	log.Printf("connecting to %s", u.String())
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	u = url.URL{Scheme: "ws", Host: "localhost:1337", Path: "/ws-events"}
	log.Printf("connecting to %s", u.String())
	cevent, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer cevent.Close()

	var tabId int64 = -1

	go func() {
		// pp.Println("create tab fuck fuck ")
		c.WriteJSON(map[string]interface{}{
			"type": "tabs.create",
			"createProperties": map[string]interface{}{
				// "url": "https://ctf.nikitastupin.com/pp/known.html?__proto__[polluted]=test",
				"url": "https://ya.ru",
				// "url": "https://moik.qa-dc.ru",
				// "url": "https://github.com",
			},
			"_timeout": 5,
		})
		var resp map[string]interface{}
		// pp.Println("create tabzzzzz")
		if err := c.ReadJSON(&resp); err != nil {
			log.WithError(err).Error("ReadJSON")
			return
		}
		// pp.Println("yeah")
		// pp.Println(resp["result"])
		// pp.Println("tab created", int64(resp["result"].(map[string]interface{})["id"].(float64)))
		atomic.StoreInt64(&tabId, int64(resp["result"].(map[string]interface{})["id"].(float64)))
	}()

	for {
		var msg map[string]interface{}
		if err := cevent.ReadJSON(&msg); err != nil {
			log.WithError(err).Error("ReadJSON")
			return
		}
		// pp.Println("eventdddddddddddddddddddddddddddddddddddddddddddddddddddddddd", msg)
		atomic.LoadInt64(&tabId)
		status := msg["tab"].(map[string]interface{})["status"].(string)
		eventTabId := int64(msg["tab"].(map[string]interface{})["id"].(float64))
		// pp.Println(tabId, eventTabId, status)
		if tabId == eventTabId && status == "complete" {

			cevent.Close()
			break
		}
	}

	c.WriteJSON(map[string]interface{}{
		"type": "dom_eval",
		// "type": "tab_create",
		"tabId": tabId,
		"code": `
			// alert("213_____hahaha");
		    // return new XMLSerializer().serializeToString(document)
			return polluted
			// return {"kdjkdj":1};
			// return "kldjkldjf";
		`,
		"_timeout": 3,
	})
	var msg map[string]interface{}
	c.ReadJSON(&msg)
	// pp.Println(msg)

	pp.Println("tabs.remove")
	atomic.LoadInt64(&tabId)
	c.WriteJSON(map[string]interface{}{
		"type":     "tabs.remove",
		"tabId":    tabId,
		"_timeout": 5,
	})
	var resp map[string]interface{}
	if err := c.ReadJSON(&resp); err != nil {
		log.WithError(err).Error("ReadJSON")
		return
	}

	pp.Println("_____THE__END______")
	// Cleanly close the connection by sending a close message and then
	// waiting (with timeout) for the server to close the connection.
	// err = c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	// if err != nil {
	// 	log.Println("write close:", err)
	// 	return
	// }
}

func blabla() {
	u := url.URL{Scheme: "ws", Host: "localhost:1337", Path: "/ws-api"}
	log.Printf("connecting to %s", u.String())
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()
	c.WriteJSON(map[string]interface{}{
		"type": "tabs.query",
		// "queryInfo": map[string]interface{}{"active": true, "currentWindow": true},
		"queryInfo": map[string]interface{}{},
		// "type": "tab_create",
		"_timeout": 8,
	})

	var resp map[string]interface{}
	c.ReadJSON(&resp)
}

// func main() {
// 	for i := 0; i < 8; i++ {
// 		go doit()
// 	}
//
// 	select {}
// }
