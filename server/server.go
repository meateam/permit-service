package server

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	ilogger "github.com/meateam/elasticsearch-logger"
	pb "github.com/meateam/permit-service/proto"
	"github.com/meateam/permit-service/service"
	"github.com/meateam/permit-service/service/mongodb"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.elastic.co/apm/module/apmgrpc"
	"go.elastic.co/apm/module/apmmongo"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

const (
	envPrefix                          = "pmts"
	configPort                         = "port"
	configHealthCheckInterval          = "health_check_interval"
	configElasticAPMIgnoreURLS         = "elastic_apm_ignore_urls"
	configMongoConnectionString        = "mongo_host"
	configMongoClientConnectionTimeout = "mongo_client_connection_timeout"
	configMongoClientPingTimeout       = "mongo_client_ping_timeout"
	configSpikeService                 = "spike_service"
	configGrantType                    = "grant_type"
	configAudience                     = "audience"
	configApprovalUrl                  = "approval_url"
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
	viper.SetDefault(configMongoConnectionString, "mongodb://localhost:27017/permit")
	viper.SetDefault(configMongoClientConnectionTimeout, 10)
	viper.SetDefault(configSpikeService, "spike-service:8080")
	viper.SetDefault(configMongoClientPingTimeout, 10)
	viper.SetDefault(configGrantType, "client_credentials")
	viper.SetDefault(configAudience, "kartoffel")
	viper.SetDefault(configApprovalUrl, "approval:8080")
	viper.SetEnvPrefix(envPrefix)
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
// Configure using environment variables.
// `HEALTH_CHECK_INTERVAL`: Interval to update serving state of the health check server.
// `PORT`: TCP port on which the grpc server would serve on.
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

	// Initiate services gRPC connections.
	spikeConn, err := initServiceConn(viper.GetString(configSpikeService))
	if err != nil {
		logger.Fatalf("couldn't setup spike service connection: %v", err)
	}

	// Create a permit service and register it on the grpc server.
	permitService := service.NewService(
		controller, logger, spikeConn, viper.GetString(configGrantType),
		viper.GetString(configAudience), viper.GetString(configApprovalUrl))
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

// serverLoggerInterceptor configures the logger interceptor for the permit server.
func serverLoggerInterceptor(logger *logrus.Logger) []grpc.ServerOption {
	// Create new logrus entry for logger interceptor.
	logrusEntry := logrus.NewEntry(logger)

	ignorePayload := ilogger.IgnoreServerMethodsDecider(
		append(
			strings.Split(viper.GetString(configElasticAPMIgnoreURLS), ","),
		)...,
	)

	ignoreInitialRequest := ilogger.IgnoreServerMethodsDecider(
		strings.Split(viper.GetString(configElasticAPMIgnoreURLS), ",")...,
	)

	// Shared options for the logger, with a custom gRPC code to log level function.
	loggerOpts := []grpc_logrus.Option{
		grpc_logrus.WithDecider(func(fullMethodName string, err error) bool {
			return ignorePayload(fullMethodName)
		}),
		grpc_logrus.WithLevels(grpc_logrus.DefaultClientCodeToLevel),
	}

	return ilogger.ElasticsearchLoggerServerInterceptor(
		logrusEntry,
		ignorePayload,
		ignoreInitialRequest,
		loggerOpts...,
	)
}

func connectToMongoDB(connectionString string) (*mongo.Client, error) {
	// Create mongodb client
	mongoOptions := options.Client().ApplyURI(connectionString).SetMonitor(apmmongo.CommandMonitor())
	mongoClient, err := mongo.NewClient(mongoOptions)
	if err != nil {
		return nil, fmt.Errorf("failed creating mongodb client with connection string %s : %v", connectionString, err)
	}

	// Connect client to mongodb
	mongoClientConnectionTimeout := viper.GetDuration(configMongoClientConnectionTimeout)
	connectionTimeoutCtx, cancelConn := context.WithTimeout(context.TODO(), mongoClientConnectionTimeout*time.Second)
	defer cancelConn()
	err = mongoClient.Connect(connectionTimeoutCtx)
	if err != nil {
		return nil, fmt.Errorf("failed connecting to mongodb with connection string %s : %v", connectionString, err)
	}

	// Check the connection
	mongoClientPingTimeout := viper.GetDuration(configMongoClientPingTimeout)
	pingTimeoutCtx, cancelPing := context.WithTimeout(context.TODO(), mongoClientPingTimeout*time.Second)
	defer cancelPing()
	err = mongoClient.Ping(pingTimeoutCtx, readpref.Primary())
	if err != nil {
		return nil, fmt.Errorf("failed pinging to mongodb with connection string %s : %v", connectionString, err)
	}

	return mongoClient, nil
}

func initMongoDBController(connectionString string) (service.Controller, error) {
	mongoClient, err := connectToMongoDB(connectionString)
	if err != nil {
		return nil, err
	}

	db, err := getMongoDatabaseName(mongoClient, connectionString)
	if err != nil {
		return nil, err
	}

	controller, err := mongodb.NewMongoController(db)
	if err != nil {
		return nil, fmt.Errorf("failed creating mongo store: %v", err)
	}

	return controller, nil
}

func getMongoDatabaseName(mongoClient *mongo.Client, connectionString string) (*mongo.Database, error) {
	connString, err := connstring.Parse(connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed parsing connection string %s: %v", connectionString, err)
	}

	return mongoClient.Database(connString.Database), nil
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

// initServiceConn creates a gRPC connection to url, returns the created connection
// and nil err on success. Returns non-nil error if any error occurred while
// creating the connection.
func initServiceConn(url string) (*grpc.ClientConn, error) {
	conn, err := grpc.Dial(url,
		grpc.WithUnaryInterceptor(apmgrpc.NewUnaryClientInterceptor()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(10<<20)),
		grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	return conn, nil
}
