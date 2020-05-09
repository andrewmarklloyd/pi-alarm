$(document).ready(function(){
  $(function() {
   $('#arm').click(function(e) {
     $.ajax({
         url: '/status',
         type: 'post',
         data: '{"armed": true}',
         success :function(response){
           console.log('response', response)
         }
     });
    });
    $('#disarm').click(function(e) {
      $.ajax({
          url: '/status',
          type: 'post',
          data: '{"armed": false}',
          success :function(response){
            console.log('response', response)
          }
      });
     });
  });
});

window.addEventListener("load", function(evt) {
  setupWebSocket()
});

function armedHandler(armed) {
  console.log("Armed: " + armed);
}

function statusHandler(status) {
  console.log("Status: " + status);
}

function setupWebSocket(){
  this.ws = new WebSocket(`ws://${location.host}/ws`);
  this.ws.onclose = function(){
    setTimeout(setupWebSocket, 1000);
  }
  this.ws.onopen = function(evt) {
    setInterval(() => {
      ws.send(JSON.stringify({ message: "ping" }));
    }, 5000)
  }
  this.ws.onmessage = function(evt) {
    var data = JSON.parse(evt.data)
    switch (data.type) {
      case "armed":
        armedHandler(data.value)
        break;
      case "status":
        statusHandler(data.value)
        break;
      default:
        console.log("Unknown data type", data.type)
    }
  }
  this.ws.onerror = function(evt) {
    console.log("Websocket error: " + evt.data);
  }
}
