// package: sessionsource
// file: sessionsource.proto

var sessionsource_pb = require("./sessionsource_pb");
var grpc = require("@improbable-eng/grpc-web").grpc;

var SessionSource = (function () {
  function SessionSource() {}
  SessionSource.serviceName = "sessionsource.SessionSource";
  return SessionSource;
}());

SessionSource.StreamOpenSessions = {
  methodName: "StreamOpenSessions",
  service: SessionSource,
  requestStream: false,
  responseStream: true,
  requestType: sessionsource_pb.StreamOpenSessionsRequest,
  responseType: sessionsource_pb.OpenSessions
};

exports.SessionSource = SessionSource;

function SessionSourceClient(serviceHost, options) {
  this.serviceHost = serviceHost;
  this.options = options || {};
}

SessionSourceClient.prototype.streamOpenSessions = function streamOpenSessions(requestMessage, metadata) {
  var listeners = {
    data: [],
    end: [],
    status: []
  };
  var client = grpc.invoke(SessionSource.StreamOpenSessions, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onMessage: function (responseMessage) {
      listeners.data.forEach(function (handler) {
        handler(responseMessage);
      });
    },
    onEnd: function (status, statusMessage, trailers) {
      listeners.status.forEach(function (handler) {
        handler({ code: status, details: statusMessage, metadata: trailers });
      });
      listeners.end.forEach(function (handler) {
        handler({ code: status, details: statusMessage, metadata: trailers });
      });
      listeners = null;
    }
  });
  return {
    on: function (type, handler) {
      listeners[type].push(handler);
      return this;
    },
    cancel: function () {
      listeners = null;
      client.close();
    }
  };
};

exports.SessionSourceClient = SessionSourceClient;

