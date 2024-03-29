# WHAT IS THE EFK STACK ?

> EFK stack is a solution that will continuously ship all the logs to a central place and a centralized view on top of it, so that we can get the logs on-demand quickly and efficiently.

|     TOOLS     | DESCRIPTION |
| ------------- | ----------- |
| Elasticsearch | For Search            |
| Fluentbit     | Log Shipper           |
| Kibana        | Visualization Tool    |


## Visual of EFK Stack

![N|Solid](efk.png)


## Why Fluentbit only ? And not others ?

> Yes, there are multiple other tools that can be used for the same purpose like Logstash, Fluentd, Filebeat, etc... But we are preferring    Fluentbit here as it provides all the functionality we need for our logging use case, plus it is really lightweight with great performance and is built on top of the best ideas of Fluentd architecture and general design.


## Charts Installation

### Elasticsearch Settings and Installation

> In values.yaml basic authentication should be enabled and creating secret is important under templates for this chart.

> values.yaml configuration: (Service port should be NodePort if we want to collect logs from other clusters nodes)
 
```
esConfig:
  elasticsearch.yml: |
    xpack.security.enabled: true


extraEnvs:
  - name: ELASTIC_PASSWORD
    valueFrom:
      secretKeyRef:
        name: elastic-credentials
        key: password
  - name: ELASTIC_USERNAME
    valueFrom:
      secretKeyRef:
        name: elastic-credentials
        key: username


enabled: true
  labels: {}
  labelsHeadless: {}
  type: NodePort
  # Consider that all endpoints are considered "ready" even if the Pods themselves are not
  # https://kubernetes.io/docs/reference/kubernetes-api/service-resources/service-v1/#ServiceSpec
  publishNotReadyAddresses: false
  nodePort: "30036"
  annotations: {}
  httpPortName: http
  transportPortName: transport
  loadBalancerIP: ""
  loadBalancerSourceRanges: []
  externalTrafficPolicy: ""

```


> Secret.yaml configuration under templates in elasicsearch charts:

```
apiVersion: v1
kind: Secret
metadata:
  name: elastic-credentials
type: opaque
data:
  username: ""
  password: ""
    
```

> apply helm chart for elasticsearch:

```
helm install elasticsearch elasticsearch -n logging
```


### Kibana Settings and Installation

> In kibana helm chart we will use the same secret that we already created in elasticsearch to login kibana UI.

> values.yaml changes

```
elasticsearchHosts: "http://elasticsearch-master:9200"
```

```
extraEnvs:
  - name: "NODE_OPTIONS"
    value: "--max-old-space-size=1800"
  - name: ELASTICSEARCH_USERNAME
    valueFrom:
      secretKeyRef:
        key: username
        name: elastic-credentials
  - name: ELASTICSEARCH_PASSWORD
    valueFrom:
      secretKeyRef:
        key: password
        name: elastic-credentials
```

```

```

> Creating ingress for accessing kibana from outside

```


```

```
helm install kibana kibana -n logging
```


### Fluentbit Settings and Installation

> Logs will go to ES as JSON docs, and ES maintains a mapping for each index. So, if you are using a single index, then if any new document comes with a field with the same name, but if the type is different than what is already saved in the index field mapping, then those log events will get dropped.

> We can use a Lua script and a simple Lua filter for adding the index field in the log event itself and then use that field in the output plugin in the Logstash_Prefix_Key

```
luaScripts:
  setIndex.lua: |
    function set_index(tag, timestamp, record)
        index = "somePrefix-"
        if record["kubernetes"] ~= nil then
            if record["kubernetes"]["namespace_name"] ~= nil then
                if record["kubernetes"]["container_name"] ~= nil then
                    record["es_index"] = index
                        .. record["kubernetes"]["namespace_name"]
                        .. "-"
                        .. record["kubernetes"]["container_name"]
                    return 1, timestamp, record
                end
                record["es_index"] = index
                    .. record["kubernetes"]["namespace_name"]
                return 1, timestamp, record
            end
        end
        return 1, timestamp, record
    end
config:
  service: |
  ...
  ...
  inputs: |
    [INPUT]
        Name tail
        Path /var/log/containers/*.log
        multiline.parser docker, cri
        Tag kube.*
        Mem_Buf_Limit 5MB
        Skip_Long_Lines On
  filters: |
    [FILTER]
        Name kubernetes
        Match kube.*
        Merge_Log On
        Keep_Log Off
        K8S-Logging.Parser On
        K8S-Logging.Exclude On
    [FILTER]
        Name lua
        Match kube.*
        script /fluent-bit/scripts/setIndex.lua
        call set_index
  outputs: |
    [OUTPUT]
        Name es
        Match kube.*
        Host elasticsearch-master
        Logstash_Format On
        Logstash_Prefix applogs # used in case prefix key is absent
        Logstash_Prefix_Key es_index
        Retry_Limit False

```

> Now, let’s see what this script and filter are doing. Nothing much, we specify a dynamic ES output index in each of the log events. As per this script, this format is being used for the index name:

```
somePrefix-{k8s_namespace_name}-{container_name}-{Date}
```

> And this somePrefix can be your company name, or whatever you like. You can choose a different format as well — just need to change the script, that’s all.

```
helm install fluent-bit fluent-bit -n logging
```


### Elasticsearch Policies

> You can specify a single ILM policy for all your indices, or you can choose a different one for all of them, but a basic Deletion policy is a must. Or if there are any compliance requirements to keep the logs for longer durations, you can check out backup options as well. But for now, let’s check how to apply a single ILM policy for all the indices.

> You need to create the ILM policy first — you can do it via the UI, from the Dev Tools (both from Kibana UI), or by sending a curl request to the ES endpoint.

> This can serve as a default Policy for all the log indices:

```
PUT _ilm/policy/retention-3day
{
  "policy": {
    "phases": {
      "hot": {
        "min_age": "0ms",
        "actions": {
          "rollover": {
            "max_primary_shard_size": "50gb",
            "max_age": "1d"
          },
          "set_priority": {
            "priority": 100
          }
        }
      },
      "warm": {
        "min_age": "1d",
        "actions": {
          "forcemerge": {
            "max_num_segments": 2
          },
          "shrink": {
            "number_of_shards": 1
          }
        }
      },
      "delete": {
        "min_age": "3d",
        "actions": {
          "delete": {
            "delete_searchable_snapshot": true
          }
        }
      }
    }
  }
}
```

> There is a Cold Phase as well which you can check, plus you can figure out the parameters as per your requirement. Now that the ILM policy is created, you need to apply this to all your indices. For that, first, we need a component template (partial settings that you can apply to multiple index templates).

```
PUT _component_template/3day-lifecyle-settings
{
  "template": {
    "settings": {
      "index": {
        "lifecycle": {
          "name": "retention-3day"
        }
      }
    }
  },
  "_meta": {
    "description": "Settings for ILM"
  }
}

```

> Then, finally, we need to create the index template specifying this component template.

```
PUT _index_template/applogs-template
{
  "index_patterns": [
    "somePrefix*"
  ],
  "composed_of": [
    "3day-lifecyle-settings"
  ],
  "priority": 500,
  "_meta": {
    "description": "Template for all app logs"
  },
  "data_stream": {
    "hidden": false
  }
}
```


### Creating Indexes on Kibana

> After logged in to Kibana you will see that indexes will be registered on Kibana with the format we created in fluentbit configuration and to see the logs coming from the nodes that we installed fluentbit on we should save the log format on Kibana.

> on Kibana click:
```
Management > Stack Management 
```

> Under Stack Management click: (you will see registered indexes which are coming from other nodes)
```
Index Management
```
![N|Solid](index-management.png)


> On the same page under Management > Kibana you will see Index Patterns

```
Index Patterns
```

![N|Solid](index-patterns.png)

> Create the indexes like that

```
logstash-*
node-*
```

> So you can see the logs in this format when you click Discover under Analytics menu:

```
Analytics > Discover
```
![N|Solid](logs.png)


















