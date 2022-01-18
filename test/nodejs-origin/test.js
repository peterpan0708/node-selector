const {
    getMiningUpdates
} = require('./rpc');

(async () => {
    try {
        getMiningUpdates('139.196.143.4');
    } catch (err) {
        console.log('err---');
        console.log(err.message);
    }
})();