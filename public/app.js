$(document).ready(function(){
  $(function() {
   $('#arm').click(function(e) {
     $.ajax({
         url: '/status',
         type: 'post',
         data: '{"armed": true}',
         success :function(response){
           armedHandler(JSON.parse(response).armed)
         }
     });
    });
    $('#disarm').click(function(e) {
      $.ajax({
          url: '/status',
          type: 'post',
          data: '{"armed": false}',
          success :function(response){
            armedHandler(JSON.parse(response).armed)
          }
      });
     });
     $('#shutdown').click(function(e) {
       res = confirm('Are you sure you want to shutdown?')
       if (res) {
         $.ajax({
             url: '/system',
             type: 'post',
             data: '{"operation": "shutdown"}',
             success :function(response){
               console.log(response)
             }
         });
       }
      });
      $('#reboot').click(function(e) {
        res = confirm('Are you sure you want to reboot?')
        if (res) {
          $.ajax({
              url: '/system',
              type: 'post',
              data: '{"operation": "reboot"}',
              success :function(response){
                console.log(response)
              }
          });
        }
       });
       $('#check-updates').click(function(e) {
         $.ajax({
             url: '/system',
             type: 'post',
             data: '{"operation": "check-updates"}',
             success :function(response){
               console.log(response)
             }
         });
         alert("Checking for updates, will restart if new version is available.")
        });
  });
});

window.addEventListener("load", function(evt) {
  setupWebSocket()
});

function armedHandler(armed) {
  if (armed === true) {
    $('#armed').html("<h4 class=\"alert-heading\">System Armed</h4>");
    $('#armed').removeClass("alert-warning");
    $('#armed').addClass("alert-success");
  } else if (armed === false) {
    $('#armed').html("<h4 class=\"alert-heading\">System Disarmed</h4>");
    $('#armed').removeClass("alert-success");
    $('#armed').addClass("alert-warning");
  } else {
    $('#armed').html("<h4 class=\"alert-heading\">System Arming Unknown</h4>");
    $('#armed').removeClass("alert-warning");
    $('#armed').removeClass("alert-success");
    $('#armed').addClass("alert-warning");
  }
}

function statusHandler(status) {
  switch (status) {
    case "OPEN":
      $('#status').html("<h4 class=\"alert-heading\">Door Open</h4>");
      $('#status').removeClass("alert-warning");
      $('#status').removeClass("alert-success");
      $('#status').addClass("alert-danger");
      break;
    case "CLOSED":
      $('#status').html("<h4 class=\"alert-heading\">Door Closed</h4>");
      $('#status').removeClass("alert-warning");
      $('#status').removeClass("alert-danger");
      $('#status').addClass("alert-success");
      break;
    case "UNKNOWN":
    default:
      $('#status').html("<h4 class=\"alert-heading\">Door Status Unknown</h4>");
      $('#status').removeClass("alert-success");
      $('#status').removeClass("alert-danger");
      $('#status').addClass("alert-warning");
  }
}

function setupWebSocket(){
  if (location.protocol == "https:") {
    this.ws = new WebSocket(`wss://${location.host}/ws`);
  } else {
    this.ws = new WebSocket(`ws://${location.host}/ws`);
  }
  this.ws.onclose = function(){
    statusHandler("UNKNOWN")
    armedHandler("UNKNOWN")
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
