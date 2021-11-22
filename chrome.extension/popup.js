let DRIVERURL_DEFAULT = "ws://localhost:1337/ws-chrome"

// In-page cache of the user's options
const options = {};

// Initialize the form with the user's option settings
chrome.storage.sync.get('options', (data) => {
  Object.assign(options, data.options);
  console.log(options);
  optionsForm.driverurl.value = options.driverurl || DRIVERURL_DEFAULT;
});

optionsForm.driverurl.addEventListener('input', (event) => {
  console.log(event);
  options.driverurl = event.target.value;
  console.log(options);
  chrome.storage.sync.set({options});
});
