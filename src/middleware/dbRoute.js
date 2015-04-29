/**
 * Created by i.n. on 24.04.2015.
 */

var log = require('./../lib/log')(module),
    DB = require('./../lib/mongoDrv')
    ;

var API = {
    setLocalVariables: function (id, variables, collectionName, cb) {
        try {
            var collection = DB.getCollection(collectionName),
                q =  {
                    "$set": {
                        "variables": variables
                    }
                };

            collection.update({"_id": id}, q, cb);

        } catch (e) {
            log.error(e['message']);
        }
    },
    
    setDomainVariables: function (domain, variables, cb) {
        
    }
};

module.exports = API;