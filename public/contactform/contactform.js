$(function() {
  $('.contactForm').on('submit', function(e) {
    e.preventDefault();
    $('#sendmessage').show();
    setTimeout(function() { $('#sendmessage').hide(); }, 3000);
    return false;
  });
});
