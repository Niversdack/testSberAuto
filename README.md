# main.go
HTTP-cервис который проверяет правильны ли пары и порядки в скобках и исправляет их.

##### GET url:8080/validate && url:8080/fix
BodyRequest:

`{
    "s":"string"
}`

Response:

`{
"v":"string",
"err":"string"
}`


##### GET url:8080/metrics
Response metrics prometheus
added customs:

`myapp_processed_ops_total`
`myapp_total_requests`

# Design architecture

### spec.json - спецификация API

[Схема сервисов и клиентский сценарий](
https://miro.com/welcomeonboard/WstG93ER9qvPN0ZEoRWl0xMqe45CaKnQ9CjY6MPPS2hCTIcySinVaMa4MBhMQUzp 
"Miro Scheme"
)



