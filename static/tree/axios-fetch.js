const AxiosFetch = {};
(function(exports){
    const http = axiosRateLimit(axios.create(),{maxRequests:1,perMilliseconds:250});
    function fetchData(url){
        const rootPromise = new Promise(async (rootResolve,rootReject)=>{     
            try{
                const response = await http.get(url);
                rootResolve(response);
            }catch(error){
                if(error.response.status==429){
                    // Convert Retry after to seconds (default to 2 seconds)
                    const delay = parseInt(error.response['retry-after'])*1000 || 2000;
                    // Retry fetch after delay
                    const promise = new Promise(function(resolve,reject){
                        setTimeout(function(){
                            let data = fetchData(url);
                            resolve(data);
                        },delay);
                    });
                    let responseData = await promise;
                    rootResolve(responseData);
                }else{
                    rootReject(error.response);
                }
            }
        });
        return rootPromise
    }
    exports.fetchData = fetchData;
})(AxiosFetch);