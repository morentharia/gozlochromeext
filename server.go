package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/k0kubun/pp"
	"github.com/lithammer/shortuuid"
	"github.com/morentharia/gozlochromeext/utils"
	log "github.com/sirupsen/logrus"
)

type M map[string]interface{}

const (
	writeWait = 20 * time.Second
	readWait  = 20 * time.Second
)

var (
	chromeSend chan M
	window     *sync.Map
	subs       *sync.Map
)

func init() {
	// log.SetLevel(log.DebugLevel)
	log.SetLevel(log.InfoLevel)
	// log.SetLevel(log.FatalLevel)
	chromeSend = make(chan M)
	window = &sync.Map{}
	subs = &sync.Map{}
}

func chromeAPIsend(ctx context.Context, msg M) (M, error) {
	if _, ok := msg["_timeout"]; ok == true {
		if timeout, ok := msg["_timeout"].(float64); ok {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, time.Duration(int(timeout))*time.Second)
			defer cancel()
		}
	}
	msg = utils.CopyMap(msg)
	out := make(chan M)

	uuid := shortuuid.New()
	window.Store(uuid, out)
	defer window.Delete(uuid)

	msg["_id"] = uuid
	log.WithField("msg", fmt.Sprintf("%#v", msg)).Debug("chromeAPI send")
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case chromeSend <- msg:
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case v, ok := <-out:
		if ok {
			log.WithField("msg", fmt.Sprintf("%#v", v)).Debug("chromeAPI recv")
			return v, nil
		}
		return nil, errors.New("WTF")
	}
}

func routeAPImsg(ctx context.Context, msg M) error {
	if _, ok := msg["_id"]; ok == true {
		if _id, ok := msg["_id"].(string); ok {
			if value, ok := window.Load(_id); ok == true {
				window.Delete(_id)
				delete(msg, "_id")
				select {
				case <-ctx.Done():
					return ctx.Err()
				case value.(chan M) <- msg:
				case <-time.After(4 * time.Second):
					log.Infof("window[%s] <- msg unreachable", _id)
				}
			}
		}
	}
	return nil
}

func routeEvents(ctx context.Context, msg M) error {
	if _, ok := msg["type"]; ok == true {
		if typ, ok := msg["type"].(string); ok {
			if typ == "event" {
				subs.Range(func(key, value interface{}) bool {
					select {
					case key.(chan M) <- msg:
					case <-time.After(4 * time.Second):
						log.Debug("subs.Delete(key)")
						subs.Delete(key)
					}
					return true
				})
			}
		}
	}
	return nil
}

func debugInfo() {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			pp.Println("subs--------")
			subs.Range(func(key, value interface{}) bool {
				pp.Println(key)
				return true
			})
			pp.Println("--------")
			pp.Println("window--------")
			window.Range(func(key, value interface{}) bool {
				pp.Println(key)
				return true
			})
			pp.Println("--------")
		}
	}
}

func server(addr string) {
	router := mux.NewRouter()
	utils.AttachProfiler(router)
	srv := &http.Server{
		Handler:      router,
		Addr:         addr,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	// go debugInfo()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lockChrome := make(chan struct{}, 1)
	router.HandleFunc("/ws-chrome", func(w http.ResponseWriter, r *http.Request) {
		select {
		case lockChrome <- struct{}{}:
		default:
			log.Error("Only one chrome connection allowed")
			http.Error(w, "Could not open websocket connection", http.StatusNotAcceptable)
			return
		}
		defer func() {
			<-lockChrome
		}()

		conn, err := websocket.Upgrade(w, r, w.Header(), 1024, 1024)
		if err != nil {
			http.Error(w, "Could not open websocket connection", http.StatusBadRequest)
			return
		}
		defer conn.Close()
		log := log.WithField("remouteAddr", conn.RemoteAddr().String()).WithField("path", "/ws-chome")

		log.Infof("chrome connected    %s/ws-chrome", conn.LocalAddr())
		defer log.Infof("chrome disconnected %s/ws-chrome", conn.LocalAddr())

		go func() {
			conn.SetReadDeadline(time.Now().Add(time.Second * 60))
			conn.SetPongHandler(func(string) error {
				log.Debug("pong")
				conn.SetReadDeadline(time.Now().Add(time.Second * 60))
				return nil
			})
			for {
				var msg M
				err := conn.ReadJSON(&msg)
				if err != nil {
					log.WithError(err).Error("ReadJSON")
					return
				}
				routeAPImsg(ctx, msg)
				routeEvents(ctx, msg)
			}
		}()

		ticker := time.NewTicker(time.Second * 5)
		defer ticker.Stop()
		for {
			select {
			case msg := <-chromeSend:
				conn.SetWriteDeadline(time.Now().Add(time.Second * 60))
				if err := conn.WriteJSON(msg); err != nil {
					log.WithError(err).Error("WriteJSON")
					return
				}
			case <-ticker.C:
				log.Debug("ping")
				conn.SetWriteDeadline(time.Now().Add(time.Second * 60))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					log.WithError(err).Error("WritePingMessage")
					return
				}
			}
		}
	})

	router.HandleFunc("/ws-events", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		select {
		case lockChrome <- struct{}{}:
			http.Error(w, "chrome is not connected", http.StatusNotAcceptable)
			<-lockChrome
			return
		default:
			fmt.Printf("%s\n", "chrome connected go go go")
		}

		conn, err := websocket.Upgrade(w, r, w.Header(), 1024, 1024)
		if err != nil {
			http.Error(w, "Could not open websocket connection", http.StatusBadRequest)
			return
		}
		defer conn.Close()
		log := log.WithField("remouteAddr", conn.RemoteAddr().String()).WithField("path", "/ws-events")

		log.Infof("event listener connected    %s/ws-events", conn.LocalAddr())
		defer log.Infof("event listener disconnected %s/ws-events", conn.LocalAddr())

		go func() {
			conn.SetReadDeadline(time.Now().Add(time.Minute * 2))
			conn.SetPongHandler(func(string) error {
				log.Debug("pong")
				conn.SetReadDeadline(time.Now().Add(time.Minute * 2))
				return nil
			})
			for {
				var msg M
				err := conn.ReadJSON(&msg)
				if err != nil {
					log.WithError(err).Error("ReadJSON")
					cancel()
					return
				}
			}
		}()

		send := make(chan M)
		subs.Store(send, true)
		defer subs.Delete(send)

		ticker := time.NewTicker(time.Second * 5)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Error("ctx.Done")
				return
			case msg := <-send:
				conn.SetWriteDeadline(time.Now().Add(time.Minute * 5))
				if err := conn.WriteJSON(msg); err != nil {
					log.WithError(err).Error("WriteJSON")
					return
				}
			case <-ticker.C:
				log.Debug("ping")
				conn.SetWriteDeadline(time.Now().Add(time.Minute * 5))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					log.WithError(err).Error("WritePingMessage")
					return
				}
			}
		}
	})

	router.HandleFunc("/ws-api", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		select {
		case lockChrome <- struct{}{}:
			http.Error(w, "chrome is not connected", http.StatusNotAcceptable)
			<-lockChrome
			return
		default:
			fmt.Printf("%s\n", "chrome connected go go go")
		}

		conn, err := websocket.Upgrade(w, r, w.Header(), 1024, 1024)
		if err != nil {
			http.Error(w, "Could not open websocket connection", http.StatusBadRequest)
			return
		}
		defer conn.Close()
		log := log.WithField("remouteAddr", conn.RemoteAddr().String()).WithField("path", "/ws-api")

		log.Infof("api client connected    %s/ws-api", conn.LocalAddr())
		defer log.Infof("api client disconnected %s/ws-api", conn.LocalAddr())

		send := make(chan M)
		go func() {
			conn.SetReadDeadline(time.Now().Add(time.Second * 60))
			conn.SetPongHandler(func(string) error {
				log.Debug("pong")
				conn.SetReadDeadline(time.Now().Add(time.Second * 60))
				return nil
			})
			for {
				select {
				case <-ctx.Done():
					log.Error("ctx.Done")
					return
				default:
				}

				var msg M
				err := conn.ReadJSON(&msg)
				if err != nil {
					log.WithError(err).Error("ReadJSON")
					cancel()
					return
				}
				go func(msg M) {
					resp, err := chromeAPIsend(ctx, msg)
					if err != nil {
						select {
						case <-ctx.Done():
							log.Error("ctx.Done")
							return
						case send <- M{"status": false, "type": msg["type"], "result": err.Error()}:
						case <-time.After(3 * time.Second):
							log.WithError(err).Error("chromeAPIsend timeout")
						}
						log.WithError(err).Error("chromeAPIsend")
						return
					}
					select {
					case <-ctx.Done():
						log.Error("ctx.Done")
						return
					case send <- resp:
					case <-time.After(3 * time.Second):
						log.Error("api resp")
					}
				}(msg)

			}
		}()

		ticker := time.NewTicker(time.Second * 5)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Error("ctx.Done")
				return
			case msg := <-send:
				conn.SetWriteDeadline(time.Now().Add(time.Second * 60))
				if err := conn.WriteJSON(msg); err != nil {
					log.WithError(err).Error("WriteJSON")
					return
				}
			case <-ticker.C:
				log.Debug("ping")
				conn.SetWriteDeadline(time.Now().Add(time.Second * 60))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					log.WithError(err).Error("WritePingMessage")
					return
				}
			}
		}
	})

	srv.ListenAndServe()
}

func main() {
	addrPtr := flag.String("addr", "localhost:1337", "")
	flag.Parse()
	server(*addrPtr)
}
