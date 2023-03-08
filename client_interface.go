package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/PodYurii/playlist_module"
	"github.com/PodYurii/playlist_module/api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"log"
	"os"
	"time"
)

type CommonFuncParams struct {
	in    *bufio.Reader
	token *uint64
	c     *api.PlaylistClient
}

func ClearingStdin(CFP *CommonFuncParams) {
	_, err := CFP.in.ReadString('\n')
	if err != nil {
		log.Println(err)
		os.Exit(2)
	}
}

func LoginWindow(CFP *CommonFuncParams) {
	var a int
	fl := true
	for fl {
		fmt.Println("Please SignIn(1), SignUp(2) or Exit(-1)")
		_, err := fmt.Fscan(CFP.in, &a)
		if err != nil {
			log.Println(err)
			ClearingStdin(CFP)
		}
		if a == 1 {
			SignInClicked(CFP, &fl)
		} else if a == 2 {
			SignUpClicked(CFP)
		} else if a == -1 {
			os.Exit(0)
		}
	}
}

func SignInClicked(CFP *CommonFuncParams, fl *bool) {
	var login, pass string
	fmt.Println("Please, enter your login and password")
	_, err := fmt.Fscan(CFP.in, &login)
	if err != nil {
		log.Println(err)
		return
	}
	_, err = fmt.Fscan(CFP.in, &pass)
	if err != nil {
		log.Println(err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	response, err := (*CFP.c).SignIn(ctx, &api.AuthRequest{Login: login, Password: pass})
	if err == status.Error(codes.NotFound, "Account does not exist or password is incorrect") {
		fmt.Println("Account does not exist or password is incorrect")
		return
	}
	if err != nil {
		log.Printf("error when calling SignIn: %s", err)
	} else {
		*CFP.token = response.GetSessionToken()
		*fl = false
		fmt.Println("Sign in successfully")
	}
}

func SignUpClicked(CFP *CommonFuncParams) {
	var login, pass string
	fmt.Println("Please, enter your login and password")
	_, err := fmt.Fscan(CFP.in, &login)
	if err != nil {
		log.Println(err)
		return
	}
	_, err = fmt.Fscan(CFP.in, &pass)
	if err != nil {
		log.Println(err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = (*CFP.c).SignUp(ctx, &api.AuthRequest{Login: login, Password: pass})
	if err == status.Error(codes.InvalidArgument, "Invalid login length: must be in range between 5 and 20") {
		fmt.Println("Invalid login length: must be in range between 5 and 20")
		return
	}
	if err != nil {
		log.Printf("error when calling SignUp: %s", err)
	} else {
		fmt.Println("SignUp success")
	}
}

func MainWindow(CFP *CommonFuncParams, pl *playlist_module.Playlist) {
	var a int
	DrawInterface(pl)
	for true {
		fmt.Println("Prev(1) Play(2) Pause(3) Next(4) Add(5) Delete(6) Exit(-1)")
		_, err := fmt.Fscan(CFP.in, &a)
		if err != nil {
			log.Println(err)
			ClearingStdin(CFP)
		}
		if a == 1 {
			PrevClicked(CFP, pl)
		} else if a == 2 {
			PlayClicked(CFP, pl)
		} else if a == 3 {
			pl.Pause()
		} else if a == 4 {
			NextClicked(CFP, pl)
		} else if a == 5 {
			AddClicked(CFP, pl)
		} else if a == 6 {
			DeleteClicked(CFP, pl)
		} else if a == -1 {
			os.Exit(0)
		}
	}
}

func PlayClicked(CFP *CommonFuncParams, pl *playlist_module.Playlist) {
	if pl.Current == nil {
		fmt.Println("There is nothing to play!")
		return
	}
	if pl.DataCheck() {
		if !DownloadCall(CFP, pl) {
			return
		}
	}
	pl.Play()
}

func DownloadCall(CFP *CommonFuncParams, pl *playlist_module.Playlist) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	stream, err := (*CFP.c).DownloadTrack(ctx, &api.TokenAndId{TrackId: pl.Current.Value.(playlist_module.Track).Id, SessionToken: *CFP.token})
	if err == status.Error(codes.NotFound, "Session not found") {
		fmt.Printf("Session is expired: return to Login menu")
		LoginWindow(CFP)
	} else if err != nil {
		log.Printf("Error when calling DownloadTrack: %s", err)
		pl.ClearData()
		return false
	}
	for {
		Response, err1 := stream.Recv()
		if err1 == io.EOF {
			break
		}
		if err1 != nil {
			log.Printf("client.DownloadTrack failed: %v", err1)
			pl.ClearData()
			return false
		}
		if Response.GetChunk() == nil {
			log.Printf("Empty chunk!")
			pl.ClearData()
			return false
		}
		pl.AddChunk(Response.Chunk)
	}
	pl.UnlockData()
	return true
}

func AddClicked(CFP *CommonFuncParams, pl *playlist_module.Playlist) {
	var a string
	fmt.Println("Enter a search string")
	_, err := fmt.Fscan(CFP.in, &a)
	if err != nil {
		log.Println(err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	stream, err := (*CFP.c).ListOfTracks(ctx, &api.FindRequest{SessionToken: *CFP.token, Findstr: a})
	if err == status.Error(codes.NotFound, "Session not found") {
		fmt.Printf("Session is expired: return to Login menu")
		LoginWindow(CFP)
	} else if err != nil {
		log.Printf("Error when calling ListOfTracks: %s", err)
		return
	}
	sl := make([]playlist_module.Track, 0, 5)
	for {
		receivedTrack, err1 := stream.Recv()
		if err1 == io.EOF {
			break
		}
		if err1 != nil {
			log.Printf("client.ListFeatures failed: %v", err1)
			return
		}
		sl = append(sl, playlist_module.Track{Name: receivedTrack.Name, Duration: time.Duration(receivedTrack.Duration), Id: receivedTrack.Id})
	}
	DrawAddSong(CFP, pl, &sl)
}

func DrawAddSong(CFP *CommonFuncParams, pl *playlist_module.Playlist, sl *[]playlist_module.Track) {
	fmt.Println("---------AddSong-----------")
	for ind, el := range *sl {
		fmt.Println("(", ind, ")", el.Duration, el.Name)
	}
	fmt.Println("---------------------------")
	a := -2
	for a < -1 || a >= len(*sl) {
		fmt.Println("Choose track by index in brackets or exit to menu(-1)")
		_, err := fmt.Fscan(CFP.in, &a)
		if err != nil {
			log.Println(err)
			ClearingStdin(CFP)
		}
	}
	if a == -1 {
		return
	}
	pl.AddSong((*sl)[a])
	DrawInterface(pl)
}
func DrawInterface(pl *playlist_module.Playlist) {
	fmt.Println("---------Current-----------")
	if pl.Current == nil {
		fmt.Println("nil")
	} else {
		fmt.Println(pl.Current.Value.(playlist_module.Track).Duration, pl.Current.Value.(playlist_module.Track).Name) // format!!!
		fmt.Println("Playing status -> ", pl.PlayingStatus())
	}
	fmt.Println("---------------------------")
	if pl.List.Len() != 0 {
		el := pl.List.Back()
		for el.Prev() != nil {
			fmt.Println(el.Value.(playlist_module.Track).Name, el.Value.(playlist_module.Track).Duration)
			el = el.Prev()
		}
		fmt.Println(el.Value.(playlist_module.Track).Name, el.Value.(playlist_module.Track).Duration)
	}
	fmt.Println("---------------------------")
}

func PrevClicked(CFP *CommonFuncParams, pl *playlist_module.Playlist) {
	ch := make(chan bool)
	go func() {
		<-ch
		if !DownloadCall(CFP, pl) {
			return
		}
	}()
	if pl.Prev(ch) {
		DrawInterface(pl)
		return
	}
	fmt.Println("Cant switch to prev track!")
}

func NextClicked(CFP *CommonFuncParams, pl *playlist_module.Playlist) {
	ch := make(chan bool)
	go func() {
		<-ch
		if !DownloadCall(CFP, pl) {
			return
		}
	}()
	if pl.Next(ch) {
		DrawInterface(pl)
		return
	}
	fmt.Println("Cant switch to next track!")
}

func DeleteClicked(CFP *CommonFuncParams, pl *playlist_module.Playlist) {
	if pl.List.Len() == 0 {
		fmt.Println("List if empty")
		return
	}
	a := -2
	for a < -1 || a >= pl.List.Len() {
		fmt.Printf("Choose track from list(from 0-lower to %d-higher) or return to menu(-1)\n", pl.List.Len()-1)
		_, err := fmt.Fscan(CFP.in, &a)
		if err != nil {
			log.Println(err)
			ClearingStdin(CFP)
		}
	}
	if a == -1 {
		return
	}
	if pl.DeleteSong(a) {
		DrawInterface(pl)
		return
	}
	fmt.Println("This track is playing!")
}
