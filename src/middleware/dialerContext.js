/**
 * Created by igor on 14.05.16.
 */

'use strict';


var log = require('./../lib/log')(module),
    dialplan = require('./dialplan'),
    DEFAULT_HANGUP_CAUSE = require('../const').DEFAULT_HANGUP_CAUSE,
    CallRouter = require('./callRouter');

module.exports = function (conn, destinationNumber, globalVariable, notExistsDirection) {
    let domainName = conn.channelData.getHeader('variable_domain_name');

    dialplan.findDialerDialplan(destinationNumber, domainName, (err, res) => {
        if (err) {
            log.error(err.message);
            conn.execute('hangup', DEFAULT_HANGUP_CAUSE);
            return
        }

        if (!res || !(res._cf instanceof Array)) {
            log.error(`Not found dialer ${destinationNumber} context`);
            conn.execute('hangup', DEFAULT_HANGUP_CAUSE);
            return
        }

        if (!domainName) {
            log.error(`Not found domain ${domainName} -> ${destinationNumber} context`);
            conn.execute('hangup', DEFAULT_HANGUP_CAUSE);
            return
        }

        // TODO caller ?
        let dn = conn.channelData.getHeader('Caller-Caller-ID-Number') || destinationNumber,
            uuid = conn.channelData.getHeader('variable_uuid'),
            answeredTime = conn.channelData.getHeader('Caller-Channel-Answered-Time')
            ;
        conn.execute('set', 'webitel_direction=dialer');

        let callflow = res._cf;

        let _router = new CallRouter(conn, {
            "globalVar": globalVariable,
            "desNumber": dn,
            "chnNumber": dn,
            "timeOffset": null,
            "versionSchema": 2,
            "domain": domainName
        });

        let exec = function () {
            try {
                log.trace('Exec: %s', dn);
                _router.run(callflow);
            } catch (e) {
                log.error(e.message);
                //TODO узнать что ответить на ошибку
                conn.execute('hangup', DEFAULT_HANGUP_CAUSE);
            }
        };

        if (+answeredTime > 0) {
            log.trace(`Channel ${uuid}  answered ${answeredTime}`);
            exec();
        } else {
            log.trace(`Channel not answered, subscribe CHANNEL_ANSWER`);
            conn.subscribe('CHANNEL_ANSWER');
            conn.on('esl::event::CHANNEL_ANSWER::*', (resEsl) => {
                console.log(resEsl);
                log.trace(`On CHANNEL_ANSWER ${uuid}`);
                exec();
            });
        }
    });
};
