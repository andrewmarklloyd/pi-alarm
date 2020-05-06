$(document).ready(function(){
  $(function() {
   $('#arm').click(function(e) {
     $.ajax({
         url: '/status',
         type: 'post',
         data: '{"armed": true}',
         success :function(){
           console.log('success')
         }
     });
    });
    $('#disarm').click(function(e) {
      $.ajax({
          url: '/status',
          type: 'post',
          data: '{"armed": false}',
          success :function(){
            console.log('success')
          }
      });
     });
  });
});
