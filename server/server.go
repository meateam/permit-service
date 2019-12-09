package server

import (
	"net"
	"time"

	ilogger "github.com/meateam/elasticsearch-logger"
	pb "github.com/meateam/permit-service/proto"
	"github.com/meateam/permit-service/service"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

const (
	configPort                         = "port"
	configHealthCheckInterval          = "health_check_interval"
	configElasticAPMIgnoreURLS         = "elastic_apm_ignore_urls"
	configMongoConnectionString        = "mongodb://localhost:27017/permission"
	configMongoClientConnectionTimeout = "mongo_client_connection_timeout"
	configMongoClientPingTimeout       = "mongo_client_ping_timeout"
)

// PermitServer is a structure that holds the permit grpc server
// and its services configuration
type PermitServer struct {
	*grpc.Server
	logger              *logrus.Logger
	port                string
	healthCheckInterval int
	permitService       service.Service
}

func init() {
	viper.SetDefault(configPort, "8080")
	viper.SetDefault(configHealthCheckInterval, 3)
	viper.SetDefault(configElasticAPMIgnoreURLS, "/grpc.health.v1.Health/Check")
	viper.SetDefault(configMongoClientConnectionTimeout, 10)
	viper.SetDefault(configMongoClientPingTimeout, 10)
	viper.AutomaticEnv()
}

// Serve accepts incoming connections on the listener `lis`, creating a new
// ServerTransport and service goroutine for each. The service goroutines
// read gRPC requests and then call the registered handlers to reply to them.
// Serve returns when `lis.Accept` fails with fatal errors. `lis` will be closed when
// this method returns.
// If `lis` is nil then Serve creates a `net.Listener` with "tcp" network listening
// on the configured `TCP_PORT`, which defaults to "8080".
// Serve will return a non-nil error unless Stop or GracefulStop is called.
func (s PermitServer) Serve(lis net.Listener) {
	listener := lis
	if lis == nil {
		l, err := net.Listen("tcp", ":"+s.port)
		if err != nil {
			s.logger.Fatalf("failed to listen: %v", err)
		}

		listener = l
	}

	s.logger.Infof("listening and serving grpc server on port %s", s.port)
	if err := s.Server.Serve(listener); err != nil {
		s.logger.Fatalf(err.Error())
	}
}

// NewServer configures and creates a grpc.Server instance.
func NewServer(logger *logrus.Logger) *PermitServer {
	// If no logger is given, create a new default logger for the server.
	if logger == nil {
		logger = ilogger.NewLogger()
	}

	// Set up grpc server opts with logger interceptor.
	serverOpts := append(
		serverLoggerInterceptor(logger),
		grpc.MaxRecvMsgSize(16<<20),
	)

	// Create a new grpc server.
	grpcServer := grpc.NewServer(
		serverOpts...,
	)

	// Connect to mongodb.
	controller, err := initMongoDBController(viper.GetString(configMongoConnectionString))
	if err != nil {
		logger.Fatalf("%v", err)
	}

	// Create a permit service and register it on the grpc server.
	permitService := service.NewService(controller, logger)
	pb.RegisterPermitServer(grpcServer, permitService)

	// Create a health server and register it on the grpc server.
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

	permitServer := &PermitServer{
		Server:              grpcServer,
		logger:              logger,
		port:                viper.GetString(configPort),
		healthCheckInterval: viper.GetInt(configHealthCheckInterval),
		permitService:       permitService,
	}

	// Health check validation goroutine worker.
	go permitServer.healthCheckWorker(healthServer)

	return permitServer
}

// serverLoggerInterceptor configures the logger interceptor for the download server.
func serverLoggerInterceptor(logger *logrus.Logger) []grpc.ServerOption {
	return nil
}

func initMongoDBController(connectionString string) (service.Controller, error) {
	// mongoClient, err := connectToMongoDB(connectionString)
	// if err != nil {
	// 	return nil, err
	// }

	// db, err := getMongoDatabaseName(mongoClient, connectionString)
	// if err != nil {
	// 	return nil, err
	// }

	// controller, err := mongodb.NewMongoController(db)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed creating mongo store: %v", err)
	// }

	return nil, nil
}

// healthCheckWorker is running an infinite loop that sets the serving status once
// in s.healthCheckInterval seconds.
func (s PermitServer) healthCheckWorker(healthServer *health.Server) {
	mongoClientPingTimeout := viper.GetDuration(configMongoClientPingTimeout)
	for {
		if s.permitService.HealthCheck(mongoClientPingTimeout * time.Second) {
			healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
		} else {
			healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
		}

		time.Sleep(time.Second * time.Duration(s.healthCheckInterval))
	}
}
