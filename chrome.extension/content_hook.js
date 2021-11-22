// https://stackoverflow.com/questions/3955803/chrome-extension-get-page-variables-in-content-script
//
//
chrome.runtime.sendMessage(
    {
        type: "get_hook_text",
        url: location.href,
    },
    (response) => {
        console.log("SET HOOK " + JSON.stringify(response));
        const script = document.createElement('script');
        script.type = "text/javascript";
        script.async = false;
        script.defer = false;
        script.textContent = response.hook_text
        const head = document.createElement('head');
        head.appendChild(script);
        document.documentElement.appendChild(head);
        head.remove();
    }
);
