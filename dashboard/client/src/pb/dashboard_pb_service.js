/* eslint-disable */
// package: dashboard
// file: dashboard.proto

var dashboard_pb = require("./dashboard_pb");
var grpc = require("@improbable-eng/grpc-web").grpc;

var Dashboard = (function () {
  function Dashboard() {}
  Dashboard.serviceName = "dashboard.Dashboard";
  return Dashboard;
}());

Dashboard.AppsList = {
  methodName: "AppsList",
  service: Dashboard,
  requestStream: false,
  responseStream: false,
  requestType: dashboard_pb.AppsListRequest,
  responseType: dashboard_pb.AppsListResponse
};

Dashboard.AppsInfo = {
  methodName: "AppsInfo",
  service: Dashboard,
  requestStream: false,
  responseStream: true,
  requestType: dashboard_pb.AppsInfoRequest,
  responseType: dashboard_pb.AppsInfoResponse
};

Dashboard.AppsMetrics = {
  methodName: "AppsMetrics",
  service: Dashboard,
  requestStream: false,
  responseStream: true,
  requestType: dashboard_pb.AppsMetricsRequest,
  responseType: dashboard_pb.AppMetricsResponse
};

Dashboard.Dmesh = {
  methodName: "Dmesh",
  service: Dashboard,
  requestStream: false,
  responseStream: false,
  requestType: dashboard_pb.DmeshRequest,
  responseType: dashboard_pb.DmeshResponse
};

Dashboard.StartApp = {
  methodName: "StartApp",
  service: Dashboard,
  requestStream: false,
  responseStream: false,
  requestType: dashboard_pb.StartAppRequest,
  responseType: dashboard_pb.StartAppResponse
};

Dashboard.StopApp = {
  methodName: "StopApp",
  service: Dashboard,
  requestStream: false,
  responseStream: false,
  requestType: dashboard_pb.StopAppRequest,
  responseType: dashboard_pb.StopAppResponse
};

exports.Dashboard = Dashboard;

function DashboardClient(serviceHost, options) {
  this.serviceHost = serviceHost;
  this.options = options || {};
}

DashboardClient.prototype.appsList = function appsList(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(Dashboard.AppsList, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

DashboardClient.prototype.appsInfo = function appsInfo(requestMessage, metadata) {
  var listeners = {
    data: [],
    end: [],
    status: []
  };
  var client = grpc.invoke(Dashboard.AppsInfo, {
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

DashboardClient.prototype.appsMetrics = function appsMetrics(requestMessage, metadata) {
  var listeners = {
    data: [],
    end: [],
    status: []
  };
  var client = grpc.invoke(Dashboard.AppsMetrics, {
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

DashboardClient.prototype.dmesh = function dmesh(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(Dashboard.Dmesh, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

DashboardClient.prototype.startApp = function startApp(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(Dashboard.StartApp, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

DashboardClient.prototype.stopApp = function stopApp(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(Dashboard.StopApp, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

exports.DashboardClient = DashboardClient;

