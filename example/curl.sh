#!/bin/bash
msg='{
  "version": "4",
  "groupKey": "{}:{severity=\"page\"}",
  "truncatedAlerts": 0,
  "status": "firing",
  "receiver": "webhook",
  "groupLabels": {
    "severity": "page",
    "alertgroup": "example1"
  },
  "commonLabels": {
    "alertname": "Alert1",
    "category": "web",
    "severity": "page"
  },
  "commonAnnotations": {
    "summary": "2 is more than 1"
  },
  "externalURL": "http://example.com:9093",
  "alerts": [
    {
      "status": "firing",
      "labels": {
        "alertname": "Alert1",
        "category": "web",
        "severity": "page",
        "a": "b",
        "c": "d"
      },
      "annotations": {
        "summary": "2 is more than 1"
      },
      "startsAt": "2020-06-09T10:22:00.309791183+09:00",
      "endsAt": "0001-01-01T00:00:00Z",
      "generatorURL": "http://example.com:9090/graph?g0.expr=1+%3C+bool+2&g0.tab=1"
    }
  ]
}'

curl -XPOST --data-binary "${msg}" http://localhost:8000/v1/webhook