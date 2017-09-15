/**
 * Created by I. Navrotskyj on 19.08.17.
 */

package acr

import (
	"github.com/webitel/acr/src/pkg/esl"
	"github.com/webitel/acr/src/pkg/logger"
	"github.com/webitel/acr/src/pkg/models"
)

func defaultContext(a *ACR, c *esl.SConn, destinationNumber string) {
	domainName := c.ChannelData.Header.Get("variable_domain_name")
	callerIdNumber := c.ChannelData.Header.Get("Channel-Caller-ID-Number")

	_, err := c.SndMsg("unset", "sip_h_call", false, false)
	if err != nil {
		logger.Error("Bad unset sip_h_call: ", err)
	}

	//TODO hash bad performance over 150cps
	if callerIdNumber != "" {
		_, err = c.SndMsg("hash", "insert/spymap/${domain_name}-"+callerIdNumber+"/${uuid}", false, false)
		if err != nil {
			logger.Error("Bad hash spymap: ", err)
		}
	}

	cf := models.CallFlow{}

	cf, err = a.DB.FindExtension(destinationNumber, domainName)
	if err != nil {
		logger.Error("Call %s db error: %s", c.Uuid, err.Error())
		c.Hangup(HANGUP_NORMAL_TEMPORARY_FAILURE)
		return
	}

	if cf.Id != 0 {
		internalCall(destinationNumber, a, c, &cf)
		return
	}

	cf, err = a.DB.FindDefault(domainName, destinationNumber)
	if err != nil {
		logger.Error("Call %s db error: %s", c.Uuid, err.Error())
		c.Hangup(HANGUP_NORMAL_TEMPORARY_FAILURE)
		return
	}

	if cf.Id != 0 {
		worldCall(destinationNumber, a, c, &cf)
		return
	}

	logger.Debug("Call %s: no found default context number %s", c.Uuid, destinationNumber)
	c.Hangup(HANGUP_NO_ROUTE_DESTINATION)
}

func internalCall(destinationNumber string, a *ACR, c *esl.SConn, cf *models.CallFlow) {
	logger.Debug("Call %s is internal", c.Uuid)
	var err error

	if c.ChannelData.Header.Get("variable_webitel_direction") == "" {
		_, err = c.SndMsg("set", "webitel_direction=internal", false, false)
		if err != nil {
			logger.Error("Bad set webitel_direction: ", err)
		}
	}

	if cf.Timezone != "" {
		_, err = c.SndMsg("set", "timezone="+cf.Timezone, false, false)
		if err != nil {
			logger.Error("Bad call %s set timezone: ", c.Uuid, err)
		} else {
			logger.Debug("Call %s set timezone %s", c.Uuid, cf.Timezone)
		}
	}
	setupPickupParameters(c, cf.Number, cf.Domain)
	a.CreateCall(destinationNumber, c, cf)
}

func worldCall(destinationNumber string, a *ACR, c *esl.SConn, cf *models.CallFlow) {
	var err error
	logger.Debug("Call %s is default context %s %s", c.Uuid, cf.Name, cf.Number)

	if c.ChannelData.Header.Get("variable_webitel_direction") == "" {
		_, err = c.SndMsg("set", "webitel_direction=outbound", false, false)
		if err != nil {
			logger.Error("Bad set webitel_direction: ", err)
		}
	}

	a.CreateCall(destinationNumber, c, cf)
}

func setupPickupParameters(c *esl.SConn, userId string, domainName string) {
	_, err := c.SndMsg("export", "dialed_extension="+userId, false, false)
	if err != nil {
		logger.Error("Bad call %s export dialed_extension: ", c.Uuid, err)
	}
	//TODO hash bad performance over 150cps
	_, err = c.SndMsg("hash", "insert/"+domainName+"-call_return/"+userId+"/${caller_id_number}", false, false)
	if err != nil {
		logger.Error("Bad call %s hash call_return: ", c.Uuid, err)
	}
	_, err = c.SndMsg("hash", "insert/"+domainName+"-last_dial_ext/"+userId+"/${uuid}", false, false)
	if err != nil {
		logger.Error("Bad call %s hash last_dial_ext: ", c.Uuid, err)
	}
	_, err = c.SndMsg("hash", "insert/"+domainName+"-last_dial_ext/global/${uuid}", false, false)
	if err != nil {
		logger.Error("Bad call %s hash last_dial_ext/global: ", c.Uuid, err)
	}
}