// https://stackoverflow.com/questions/3955803/chrome-extension-get-page-variables-in-content-script
//

function EvalRequest(req) {
    let script = document.createElement("script");
    script.type = "text/javascript";
    script.async = false;
    script.defer = true;
    //todo: add timeout
    fullcode = `
        (function() {
            var AsyncFunction = Object.getPrototypeOf(async function(){}).constructor;
            let fff = AsyncFunction("` + JSON.stringify(req.code).slice(1, -1)+ `");
            fff().then(result => {
                window.postMessage(
                    {
                        _id: "` + JSON.stringify(req._id).slice(1, -1)+ `",
                        type: 'DOM_EVAL',
                        status: true,
                        result: result
                    },
                    '*'
                );
            }).catch(e => {
               console.error(e);
               window.postMessage(
                   {
                       _id: "` + JSON.stringify(req._id).slice(1, -1)+ `",
                       type: 'DOM_EVAL',
                       status: false,
                       result: e.toString()
                   },
                   '*'
               );
            });

        })();
    `
    console.log(fullcode);
    script.textContent = fullcode;
    document.body.appendChild(script).parentNode.removeChild(script);
}


window.addEventListener("message", (event) => {
    // We only accept messages from ourselves
    if (event.source != window) {
        return;
    }

    if (event.data.type && (event.data.type == "DOM_EVAL")) {
        console.log("Content script received: " + event.data);
        // port.postMessage(event.data.text);
        chrome.runtime.sendMessage({
            _id: event.data._id,
            type: "dom_eval",
            status: event.data.status,
            result: event.data.result,
        });
    }
}, false);

chrome.runtime.onMessage.addListener(function (request, _, sendResponse) {
    try {
        console.log(request);
        switch(request.type){
            case "dom_eval":
                EvalRequest(request);
                break;
            default:
                console.warn("unknown message" + request);
        }
    } catch (e) {
        console.error(e);
    }
});
