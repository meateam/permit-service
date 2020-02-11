module github.com/meateam/permit-service

go 1.12

require (
	github.com/DataDog/zstd v1.4.4 // indirect
	github.com/golang/protobuf v1.3.2
	github.com/grpc-ecosystem/go-grpc-middleware v1.0.0
	github.com/meateam/elasticsearch-logger v1.1.3-0.20190901111807-4e8b84fb9fda
	github.com/meateam/spike-service v0.0.0-20191212084848-b7760c5526b3
	github.com/segmentio/ksuid v1.0.2
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/viper v1.6.1
	github.com/tidwall/pretty v1.0.0 // indirect
	go.elastic.co/apm/module/apmgrpc v1.6.0
	go.elastic.co/apm/module/apmmongo v1.6.0
	go.mongodb.org/mongo-driver v1.2.0
	google.golang.org/grpc v1.25.1
)

replace github.com/meateam/permit-service/service => ./service

replace github.com/meateam/permit-service/proto => ./proto
