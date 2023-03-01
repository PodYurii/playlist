package main

import (
	"context"
	"fmt"
	"github.com/PodYurii/playlist_module/api"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"time"
)

type track struct {
	Name     string
	Duration int64
	Id       uint64
}

type FilePath struct {
	Id   uint64
	Path string
}

type User struct {
	Login    string
	Password string
}

type PlaylistServer struct {
	api.UnimplementedPlaylistServer
	Users    *mongo.Collection
	Sessions map[uint64]*Session
	Tracks   *mongo.Collection
	Files    *mongo.Collection
}

func NewPlaylistServer() *PlaylistServer {
	s := PlaylistServer{}
	s.Users = client.Database("playlist").Collection("Users")
	s.Tracks = client.Database("playlist").Collection("Tracks")
	s.Files = client.Database("playlist").Collection("Files")
	s.Sessions = make(map[uint64]*Session)
	return &s
}

type Session struct {
	expand *time.Timer
}

func (obj *Session) ExpandSession() {
	obj.expand.Reset(time.Minute * 30)
}

func (s PlaylistServer) CreateSession() uint64 {
	var uid uint64
	found := true
	for found && uid != 0 {
		uid = rand.Uint64()
		_, found = s.Sessions[uid]
	}
	var newObj Session
	newObj.expand = time.AfterFunc(time.Minute*30, func() { s.DeleteSession(uid) })
	s.Sessions[uid] = &newObj
	return uid
}

func (s PlaylistServer) DeleteSession(uid uint64) {
	delete(s.Sessions, uid)
}

func (s PlaylistServer) SignIn(_ context.Context, request *api.AuthRequest) (*api.OnlyToken, error) {
	//pass, found := s.Users[request.GetLogin()]
	var result User
	filter := bson.D{{"login", request.GetLogin()}}
	err := s.Users.FindOne(context.TODO(), filter).Decode(&result)
	if err == nil && result.Password == request.GetPassword() {
		return &api.OnlyToken{SessionToken: s.CreateSession()}, nil
	}
	return &api.OnlyToken{}, status.Error(codes.NotFound, "Account does not exist or password is incorrect")
}

func (s PlaylistServer) SignUp(_ context.Context, request *api.AuthRequest) (*api.Empty, error) {
	length := len(request.GetLogin())
	if length < 5 || length > 20 {
		return &api.Empty{}, status.Error(codes.Internal, "Invalid login length: must be in range between 5 and 20")
	}
	length = len(request.GetPassword())
	if length < 5 || length > 20 {
		return &api.Empty{}, status.Error(codes.Internal, "Invalid password length: must be in range between 5 and 20")
	}
	var result User
	filter := bson.D{{"login", request.GetLogin()}}
	err := s.Users.FindOne(context.TODO(), filter).Decode(&result)
	if err != nil {
		return &api.Empty{}, status.Error(codes.AlreadyExists, "This login is already taken")
	}
	//s.Users[request.GetLogin()] = request.GetPassword()
	result.Login = request.Login
	result.Password = request.Password
	_, err = s.Users.InsertOne(context.TODO(), result)
	if err != nil {
		log.Print(err)
		return &api.Empty{}, status.Error(codes.Canceled, "Server error: try it later")
	}
	return &api.Empty{}, nil
}

func (s PlaylistServer) ListOfTracks(request *api.FindRequest, stream api.Playlist_ListOfTracksServer) error {
	token := request.GetSessionToken()
	str := request.GetFindstr()
	if token == 0 {
		return status.Error(codes.InvalidArgument, "Token is equal 0")
	}
	_, found := s.Sessions[token]
	if !found {
		return status.Error(codes.NotFound, "Session not found")
	}
	opts := options.Find()
	opts.SetLimit(5)
	filter := bson.D{{"$text", bson.D{{"name", str}}}}
	cursor, err := s.Users.Find(context.TODO(), filter, opts)
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err = cursor.Close(ctx)
		if err != nil {
			log.Print(err)
		}
	}(cursor, context.TODO())
	if err != mongo.ErrNoDocuments && err != nil {
		return err
	}
	for cursor.Next(context.TODO()) {
		var el track
		err = cursor.Decode(&el)
		if err != nil {
			log.Print(err)
			continue
		}
		if err = stream.Send(&api.ListResponse{Id: el.Id, Duration: el.Duration, Name: el.Name}); err != nil {
			log.Print(err)
			return err
		}
	}
	return nil
}

func (s PlaylistServer) DownloadTrack(request *api.TokenAndId, stream api.Playlist_DownloadTrackServer) error {
	token := request.GetSessionToken()
	id := request.GetTrackId()
	if token == 0 || id == 0 {
		return status.Error(codes.InvalidArgument, "Token or id is equal 0")
	}
	_, found := s.Sessions[token]
	if !found {
		return status.Error(codes.NotFound, "Session not found")
	}
	var result FilePath
	filter := bson.D{{"Id", id}}
	err := s.Users.FindOne(context.TODO(), filter).Decode(&result)
	if err != nil {
		log.Print(err)
		return err
	}
	file, err := os.Open(result.Path)
	if err != nil {
		return err
	}
	buf := make([]byte, 1024)
	var num int
	for {
		num, err = file.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		chunk := buf[:num]
		if err = stream.Send(&api.TrackResponse{Chunk: chunk}); err != nil {
			log.Print(err)
			return err
		}
	}
	return nil
}

var client *mongo.Client

func init() {
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017/")
	var err error
	client, err = mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	// create a listener on TCP port 7777
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", "localhost", 7777))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	} // create a server instance
	s := NewPlaylistServer()
	creds, err := credentials.NewServerTLSFromFile("cert/server.crt", "cert/server.key")
	if err != nil {
		log.Fatalf("could not load TLS keys: %s", err)
	} // Create an array of gRPC options with the credentials
	opts := []grpc.ServerOption{grpc.Creds(creds)} // create a gRPC server object
	grpcServer := grpc.NewServer(opts...)          // attach the Ping service to the server
	api.RegisterPlaylistServer(grpcServer, s)      // start the server
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %s", err)
	}
}

// Server db funcs
// Sign in with existed session
