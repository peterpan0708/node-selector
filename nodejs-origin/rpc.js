const http = require('http');
/**
 * 开启一个长连接，有新块出现时会触发 'newBlock' 事件
 */
const getMiningUpdates = (my_host) => {
    const url = `http://${my_host}:1848/chainweb/0.0/mainnet01/header/updates`,
        options = {
            method: 'GET',
            headers: {
                'Connection': 'keep-alive',
                'Pragma': 'no-cache',
                'Cache-Control': 'no-cache',
                'Accept': 'text/event-stream',
                'Sec-Fetch-Site': 'same-site',
                'Sec-Fetch-Mode': 'cors',
                'Sec-Fetch-Dest': 'empty',
                'Referer': 'https://explorer.chainweb.com/mainnet',
                'Accept-Language': 'zh-CN,zh;q=0.9',
                'Transfer-Encoding': 'chunked'
            },
            rejectUnauthorized: false // add when working with https sites
            // requestCert: false, // add when working with https sites
            // agent: false, // add when working with https sites
        };
    // const x = http.request()
    const client = http.request(url, options, (message) => {
        const {statusCode, statusMessage} = message;

        if (statusCode == 200 && (statusMessage.toUpperCase() === 'OK')) {
            message.on('data', (chunk) => {
                let res = chunk.toString('utf8');
                let obj = JSON.parse(res.split('data:')[1]);

                let {header} = obj;
                let {height, chainId} = header;
                console.log(my_host, height, chainId, new Date());

                // console.log('\n');
            }).on('end', () => {
            });
        } else {
            message.on('data', (chunk) => {
            }).on('end', (chunk) => {
            });
            console.log(`https status code is ${statusCode}.message is ${statusMessage}`);
        }
    });

    client.on('error', (err) => {
    });

    client.on('close', () => {
        console.log('getMiningUpdates close');
    });

    client.end();
};

module.exports.getMiningUpdates = getMiningUpdates;