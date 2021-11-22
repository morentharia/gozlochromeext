// let g_websocket_url = "ws://localhost:1337/ws-chromezzz"
let DRIVERURL_DEFAULT = "ws://localhost:1337/ws-chrome"
let HOOK_TEXT = `console.log("default hook: do nothing")`
let g_websocket_url = "ws://todo/"

var socket = connect()

chrome.storage.sync.get('options', (data) => {
    let options = {}
    Object.assign(options, data.options);
    console.log(options);
    g_websocket_url = options.driverurl || DRIVERURL_DEFAULT;
});

// Watch for changes to the user's options & apply them
chrome.storage.onChanged.addListener((changes, area) => {
    console.log('wow', changes, area);
    if (area === 'sync' && changes.options?.newValue) {
        // const debugMode = Boolean(changes.options.newValue.debug);
        // console.log('enable debug mode?', debugMode);
        // setDebugMode(debugMode);
        g_websocket_url = changes.options.newValue.driverurl
        console.log(g_websocket_url);
        socket.close()
    }
});

// chrome.browserAction.onClicked.addListener(function(tab) {
// 	chrome.tabs.create({'url': chrome.extension.getURL('popup.html')}, function(tab) {
// 		// Tab opened.
// 	});
// });

chrome.runtime.onMessage.addListener(function(data, sender, sendResponse) {
    // console.log("data", JSON.stringify(data));
    switch (data.type) {
        case 'get_hook_text':
            sendResponse({hook_text: HOOK_TEXT});
            break;
        default:
            if (socket.readyState === 1) {
                socket.send(JSON.stringify(data));
            }
    }
});

chrome.tabs.onUpdated.addListener(function(tabId, changeInfo, tab) {
    if (socket.readyState === 1) {
        socket.send(JSON.stringify({
            "type": "event",
            "name": "tabs.onUpdated",
            "tabId": tabId,
            "changeInfo": changeInfo,
            "tab": tab
        }));
    }
});

function connect() {
    try {
        socket = new WebSocket(g_websocket_url);
    } catch (e) {
        /* handle error */
        socket = new WebSocket("ws://todoblabla/ws-chrome");
    }


    socket.onopen = function() {
        console.log("Socket is open");
    };

    socket.onmessage = function(e) {
        let data
        try {
            data = JSON.parse(e.data)
            // console.log(data.type);
            // console.log(data);
            switch (data.type) {
                case "dom_eval":
                    if (typeof data.tabId === 'undefined') {
                        chrome.tabs.query({
                            active: true,
                            currentWindow: true
                        }, function(tabs) {
                            var activeTab = tabs[0];
                            // chrome.tabs.getCurrent(function (activeTab) {
                            if (typeof activeTab === "undefined") {
                                console.error("no active tab");
                                socket.send(JSON.stringify({
                                    "_id": data._id,
                                    "type": data.type,
                                    "status": false,
                                    "result": "Erorr: no active tab"
                                }));
                                return;
                            }
                            console.log("chrome.tabs.sendMessage(", activeTab.id, ",", data, ")");
                            chrome.tabs.sendMessage(activeTab.id, data);
                        });
                    } else {
                        chrome.tabs.sendMessage(data.tabId, data);
                    }
                    break;

                case "set_hook":
                    HOOK_TEXT = data.value
                    socket.send(JSON.stringify({
                        "_id": data._id,
                        "type": data.type,
                        "status": true,
                        "result": "ok"
                    }));
                    break;

                case "tabs.create":
                    // console.log("tabs.create", data);
                    chrome.tabs.create(
                        data.createProperties,
                        function(tab) {
                            // console.log("tabs.create done", data._id);
                            socket.send(JSON.stringify({
                                "_id": data._id,
                                "type": data.type,
                                "status": true,
                                "result": tab
                            }));
                        }
                    );
                    break;

                case "tabs.get":
                    chrome.tabs.get(data.tabId, (tab) => {
                        socket.send(JSON.stringify({
                            "_id": data._id,
                            "type": data.type,
                            "status": true,
                            "result": tab
                        }));
                    });
                    break

                case "tabs.query":
                    chrome.tabs.query(data.queryInfo, (tabs) => {
                        socket.send(JSON.stringify({
                            "_id": data._id,
                            "type": data.type,
                            "status": true,
                            "result": tabs
                        }));
                    });
                    break

                case "tabs.remove":
                    chrome.tabs.remove(data.tabId, () => {
                        socket.send(JSON.stringify({
                            "_id": data._id,
                            "type": data.type,
                            "status": true,
                            "result": null,
                        }));
                    });
                    break

                case "tabs.update":
                    chrome.tabs.update(data.tabId, data.updateProperties, () => {
                        socket.send(JSON.stringify({
                            "_id": data._id,
                            "type": data.type,
                            "status": true,
                            "result": tab
                        }));
                    });
                    break

                case "ping":
                    socket.send(JSON.stringify({
                        "_id": data._id,
                        "type": data.type,
                        "status": true,
                        "result": "pong",
                    }));
                    break

                default:
                    socket.send(JSON.stringify({
                        "type": data.type,
                        "_id": data._id,
                        "status": false,
                        "result": "unknown type message",
                    }));
                    console.error(e);
                    break
            }
        } catch (exc) {
            console.error("Fuck fuck fuck: ", exc);
            socket.send(JSON.stringify({
                "type": data.type,
                "_id": data._id,
                "status": false,
                "result": "Error: " + exc,
            }));
        }
    };



    socket.onclose = function() {
        console.log("Socket closed reconnect after 1sec");
        setTimeout(function() {
            console.log("reconnect", g_websocket_url);
            connect(g_websocket_url);
        }, 1000);
    };

    socket.onerror = function(err) {
        console.error("Socket encountered error: ", err.message, "Closing socket");
        socket.close();
    };

    return socket
}


const maybeSame = (a, b) => {
    return a.toUpperCase().trim() == b.toUpperCase().trim();
};
const isCSPHeader = (header) => {
    return maybeSame(header.name, 'Content-Security-Policy');
};
const isCached = (header) => {
    return maybeSame(header.name, 'If-None-Match');
};

chrome.webRequest.onHeadersReceived.addListener((response) => {
    response.responseHeaders.forEach(header => {
        if (isCSPHeader(header)) {
            // console.log(header.value);
            header.value = `default-src * 'unsafe-inline' 'unsafe-eval' data: blob:; `;
        };
        if (isCached(header)) {
            header.name = 'lol';
        };
    });
    return {
        responseHeaders: response.responseHeaders,
    };
}, {
    urls: ['<all_urls>']
}, ['blocking', 'responseHeaders', 'extraHeaders']);
