# axiom-honeycomb-proxy
Forward/Multiplex your events to Axiom and Honeycomb

To try it start it up and point your honeycomb stuff to the deployment.

#### event requests
```curl http://localhost:3111/honeycomb/v1/events/test-via-curl -X POST \
  -H "X-Honeycomb-Team: <your-honeycomb-key>" \
  -H "X-Honeycomb-Event-Time: 2018-02-09T02:01:23.115Z" \
  -d '{"method":"GET","endpoint":"/foo","shard":"users","dur_ms":32}'
```

#### batch requests
```curl  http://localhost:3111/honeycomb/v1/batch/<dataset> -X POST \
  -H "X-Honeycomb-Team: <your-honeycomb-key>" \
  -d '[
        {
          "time":"2018-02-09T02:01:23.115Z",
          "data":{"key1":"val1","key2":"val2"}
        },
        {
          "data":{"key3":"val3"}
        }
      ]'
```

### Note
Honeycomb creates datasets when you push to them. Axiom does not support that (yet). Make sure you are create the matching datasets on the axiom-side first.