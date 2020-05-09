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
    console.log("Message: " + evt.data);
  }
  this.ws.onerror = function(evt) {
      console.log("Websocket error: " + evt.data);
  }
}
