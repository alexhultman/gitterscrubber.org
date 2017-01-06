'use strict';

const http = require('http');
const request = require('request');
const url = require('url');
const querystring = require('querystring');

// your Gitter oauth key
const key = 'put your gitter oauth key here';
// your Gitter oauth secret
const secret = 'put your gitter oauth secret here';
// your Gitter redirect aka the address of this server
const redirect = 'http://global address to this server:port/';
// port of this server
const port = 3000;

const scrubOneToOnes = true;

// click this link with ctrl + mouse click to authenticate in browser
console.log('Click this to scrub: ' + 'https://gitter.im/login/oauth/authorize?' + querystring.stringify({
  client_id: key,
  response_type: 'code',
  redirect_uri: redirect
}));

function fetchMessages(token, roomId, beforeId, cb) {
  var buffer = '';
  request.get({url: 'https://api.gitter.im/v1/rooms/' + roomId + '/chatMessages?limit=100' + (beforeId ? ('&beforeId=' + beforeId) : ''), headers: {
    'authorization': 'Bearer ' + token,
    'accept': 'application/json'
  }}).on('data', function(data) {
    buffer += data.toString();
  }).on('end', function() {
    var messages = JSON.parse(buffer);
    if (messages.error) {
      console.log('Failed, retrying in a while!');
      console.log(messages.error);
      setTimeout(function() {
        fetchMessages(token, roomId, beforeId, cb);
      }, 1000);
    } else {
      cb(messages);
    }
  });
}

function deleteMessage(token, roomId, messageId, cb) {
  request.delete({url: 'https://api.gitter.im/v1/rooms/' + roomId + '/chatMessages/' + messageId, headers: {
    'authorization': 'Bearer ' + token
  }}).on('response', function(response) {
    if (response.statusCode != 204) {
      console.log('Failed to delete, retrying in a while!');
      setTimeout(function() {
        deleteMessage(token, roomId, messageId, cb);
      }, 1000);
    } else {
      cb();
    }
  });
}

function deleteMyMessages(token, userId, roomId, messages, index, cb) {
  if (index == messages.length) {
    cb();
    return;
  }
  if (messages[index].fromUser.id === userId) {
    deleteMessage(token, roomId, messages[index].id, function() {
        console.log('Deleted message: ' + messages[index].text);
        deleteMyMessages(token, userId, roomId, messages, index + 1, cb);
    });
  } else {
    deleteMyMessages(token, userId, roomId, messages, index + 1, cb);
  }
}

function readAllMessages(token, userId, roomId, beforeId, cb) {
  fetchMessages(token, roomId, beforeId, function(messages) {
    if (messages.length) {
      console.log('Meddelanden: ' + messages.length + ' Första datum: ' + messages[0].sent);
    }

    deleteMyMessages(token, userId, roomId, messages, 0, function() {
      if (messages.length) {
        readAllMessages(token, userId, roomId, messages[0].id, cb);
      } else {
        console.log('Reached the end of this room, calling back!');
        cb();
      }
    });
  });
}

function clearRooms(token, userId, rooms, index) {
  if (index == rooms.length) {
    return;
  }
  // scrub everything, even one to one
  if (!scrubOneToOnes && rooms[index].oneToOne) {
    clearRooms(token, userId, rooms, index + 1);
  } else {
    console.log('Scrubbing all messages in room: ' + rooms[index].name);
    readAllMessages(token, userId, rooms[index].id, null, function() {
      clearRooms(token, userId, rooms, index + 1);
    });
  }
}

function getAccessToken(code, cb) {
  var buffer = '';
  request.post({url: 'https://gitter.im/login/oauth/token', body: JSON.stringify({
    client_id: key,
    client_secret: secret,
    redirect_uri: redirect,
    grant_type: 'authorization_code',
    code: code
  }), headers: {
    'content-type': 'application/json',
    'accept': 'application/json'
  }})
  .on('data', function(data) {
    buffer += data.toString();
  }).on('end', function() {
    var access = JSON.parse(buffer);
    cb(access.access_token);
  });
}

function getUser(token, cb) {
  var buffer = '';
  request.get({url: 'https://api.gitter.im/v1/user/me', headers: {
    'authorization': 'Bearer ' + token,
    'accept': 'application/json'
  }}).on('data', function(data) {
    buffer += data.toString();
  }).on('end', function() {
    var user = JSON.parse(buffer);
    cb(user.id, user.displayName);
  });
}

function getPublicRooms(token, cb) {
  var buffer = '';
  request.get({url: 'https://api.gitter.im/v1/user/me/rooms', headers: {
    'authorization': 'Bearer ' + token,
    'accept': 'application/json'
  }}).on('data', function(data) {
    buffer += data.toString();
  }).on('end', function() {
    var rooms = JSON.parse(buffer);
    cb(rooms);
  });
}

const server = http.createServer((req, res) => {
  var code = querystring.parse(url.parse(req.url).query).code; // error om deny
  if (code && code.length == 40) {
    getAccessToken(code, function(token) {
      getUser(token, function(userId, userName) {
        console.log('[Scrubbing user: ' + userName + ']');
        getPublicRooms(token, function(rooms) {
          // respond with what rooms will be deleted and who you are
          var response = '';
          response += '<h1>gitterscrubber.org</h1>';
          response += 'You are logged in as: <b>' + userName + '</b>';
          response += '<h3>Deleting your messages in public rooms:</h3>';
          for (let i = 0; i < rooms.length; i++) {
            // delete everything
            if (scrubOneToOnes || !rooms[i].oneToOne) {
              response += '<b>' + rooms[i].name + '</b><br>';
            }
          }
          response += '<p>Your task has been posted. This will take a lot of time to finish. Expect at least a day of waiting.</p>';
          res.end(response);
          // start the scrubbing
          clearRooms(token, userId, rooms, 0);
        });
      });
    });
  } else {
    res.end('Seems like you denied access');
  }
});

server.listen(port);
