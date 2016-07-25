/**
 * Created by i.navrotskyj on 17.02.2015.
 */
'use strict';

var Client = require('node-rest-client').Client,
    client = new Client(),
    //EventEmitter2 = require('eventemitter2').EventEmitter2,
    log = require('./../lib/log')(module);

var METHODS = {
    GET: "get",
    POST: "post",
    PUT: "put",
    DELETE: "delete"
};

var DEF_EXPORT_VAR = function () {
    return {
        "effective_caller_id_name": "callerIdName",
        "owner_caller_id_number": "callerIdOwner"
    };
};

var DEF_HEADERS = function () {
    return {
        "Content-Type":"application/json"
    }
};

var DEF_DATA = function() {
    return {
        "callerIdNumber": "${Caller-Caller-ID-Number}"
    };
};

client.on('error', function (err) {
    log.error(err);
});


module.exports = function (parameters, router, cb) {

    if (!parameters['url']) {
        cb(new Error('Bad request'));
        return;
    }
    var path, current;
    
    var _parseRequest = function (dataRequestLib, a, b) {
        try {
            if (typeof parameters.exportCookie == 'string' && a.headers['set-cookie'] && router.connection.socket) {
                router.__setVar({
                    "setVar": `${parameters.exportCookie}=${a.headers['set-cookie'].join(';')}`
                })
            }
            var jsonData;
            var dataRequest = Buffer.isBuffer(dataRequestLib) ? dataRequestLib.toString('utf8') : dataRequestLib;
            if (a.headers["content-type"] && !~a.headers["content-type"].indexOf('application/json')) {
                log.error(`No support response content type ${a.headers["content-type"]}`);
                return;
            }
            log.debug(dataRequest);
            if (typeof dataRequest === 'object') {
                jsonData = dataRequest;
            } else {
                jsonData = JSON.parse(dataRequest);
            }
            for (var key in exportVariables) {
                path = exportVariables[key] || '';
                current = jsonData;
                path.split('.').forEach(function(token) {
                    current = current && current[token];
                });

                if (!current) continue;
                if (router.connection.socket) {
                    router.__setVar({
                        "setVar": "all:" + key + "=" + current
                    });
                }
            }
        } catch (e) {
            log.error(e);
        } finally {
            cb();
        }
    };

    let parseObject = (o) => {
        let n = JSON.stringify(o);
        // TODO ...
        return JSON.parse(router._parseVariable(n));
    };

    var method = parameters['method'] || 'post',
        exportVariables = parameters['exportVariables'] || DEF_EXPORT_VAR(),
        headers = parameters['headers'] || DEF_HEADERS();


    var webArgs = {
        data: parameters['data'] || DEF_DATA(),
        headers: parseObject(headers),
        requestConfig: {
            timeout: 1000, //request timeout in milliseconds
            keepAlive: false //Enable/disable keep-alive functionalityidle socket.
        },
        responseConfig: {
            timeout: 1000 //response timeout
        }
    };
    method = method.toLowerCase();

    var contentType = (webArgs.headers && webArgs.headers['Content-Type']) || '';
    if (contentType.toLowerCase() == "application/x-www-form-urlencoded") {
        if (webArgs.data instanceof Object) {
            let _data = [];
            for (let key in webArgs.data) {
                _data.push(key + '=' + webArgs.data[key]);
            }
            webArgs.data = router._parseVariable(_data.join('&')).replace(/\s/g, '+');
        } else {
            webArgs.data = router._parseVariable('' + webArgs.data).replace(/\s/g, '+');
        }
    } else {
        webArgs.data = parseObject(webArgs.data);
        if (parameters.path) {
            webArgs.path = parseObject(parameters.path);
        }
    }

    // console.dir(webArgs, {depth: 5, color: true});

    var req;
    if (method == METHODS.GET) {
        webArgs.parameters = webArgs.data;
        req = client.get(parameters['url'], webArgs, _parseRequest);
    } else if (method == METHODS.POST) {
        req = client.post(parameters['url'], webArgs, _parseRequest);
    } else {
        log.error('Bad parameters method');
        cb();
        return;
    }

    req.on('requestTimeout',function(req){
        log.warn("request has expired");
        req.abort();
        cb();
    });

    req.on('error',function(err){
        log.error(err.message);
        cb();
    });

    req.on('responseTimeout',function(){
        log.warn("response has expired");
    });
};