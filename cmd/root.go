package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	cardBattle "github.com/renosyah/simpleCardBattleServer/cardBattle"
)

var (
	dbPool  *sql.DB
	cfgFile string
)

var rootCmd = &cobra.Command{
	Use: "app",
	PreRun: func(cmd *cobra.Command, args []string) {

		// init package

	},
	Run: func(cmd *cobra.Command, args []string) {

		ctx, cancel := context.WithCancel(SignalContext(context.Background()))
		defer cancel()

		lis, err := net.Listen("tcp", fmt.Sprint(":", viper.GetInt("app.port")))
		if err != nil {
			fmt.Println(fmt.Sprintf("failed to listen: %v", err))
		}

		s := grpc.NewServer()

		lobby := &cardBattle.CardBattleServer{
			Ctx: ctx,
			Lobby: (&cardBattle.Lobby{
				Broadcast:     make(chan cardBattle.LobbyStream),
				ClientStreams: make(map[string]chan cardBattle.LobbyStream),
			}).NewHub(),
			Queue: (&cardBattle.QueueRoom{
				PlayersInWaitingRoom: make(map[string]*cardBattle.PlayerWithCards),
				Broadcast:            make(chan cardBattle.QueueStream),
				ClientStreams:        make(map[string]chan cardBattle.QueueStream),
			}).NewHub(),
			Room:    make(map[string]*cardBattle.Room),
			Players: make(map[string]*cardBattle.PlayerWithCards),
			Config: &cardBattle.ServerConfig{
				AmountDefaultCash: int64(viper.GetInt("game.cash")),
				AmountDefaultCard: int32(viper.GetInt("game.card")),
				AmountDefaultExp:  int64(viper.GetInt("game.exp")),
				Level:             int32(viper.GetInt("game.level")),
				MaxReserveSlot:    int32(viper.GetInt("game.reserve")),
				MaxDeckSlot:       int32(viper.GetInt("game.deck")),
			},
			RoomConfig: &cardBattle.RoomManagementConfig{
				ExpiredTime:              int32(viper.GetInt("room.expired_time")),
				DefaultMaxPlayer:         int32(viper.GetInt("room.default_max_player")),
				DefaultMaxPlayerDeck:     int32(viper.GetInt("room.default_max_player_deck")),
				DefaultCurrentDeployment: int32(viper.GetInt("room.default_current_deployment")),
				DefaultMaxDeployment:     int32(viper.GetInt("room.default_max_deployment")),
				DefaultEachPlayerHealth:  int64(viper.GetInt("room.default_each_player_health")),
				DefaultCoolDownTime:      int32(viper.GetInt("room.default_cool_down_time")),
				DefaultMaxCardReward:     int32(viper.GetInt("room.default_max_card_reward")),
				DefaultCashReward:        int64(viper.GetInt("room.default_cash_reward")),
				DefaultExpReward:         int64(viper.GetInt("room.default_exp_reward")),
			},
			Shop: (&cardBattle.CardShop{
				Cards:         make(map[string]*cardBattle.Card),
				TotalCard:     viper.GetInt("shop.total"),
				MinLevel:      viper.GetInt("shop.min"),
				MaxLevel:      viper.GetInt("shop.max"),
				Time:          viper.GetInt("shop.time"),
				MaxItem:       viper.GetInt("shop.max_item"),
				Broadcast:     make(chan cardBattle.ShopStream),
				ClientStreams: make(map[string]chan cardBattle.ShopStream),
			}).NewHub(),
		}

		lobby.RunCardShop()

		lobby.RunRoomMaker()
		lobby.RemoveEmptyRoom()

		cardBattle.RegisterCardBattleServiceServer(s, lobby)

		reflection.Register(s)

		mux := http.NewServeMux()
		mux.Handle("/", http.FileServer(http.Dir(viper.GetString("app.dirhost"))))
		restSrv := &http.Server{
			Addr:    fmt.Sprintf(":%d", viper.GetInt("app.port_rest")),
			Handler: mux,
		}

		go func() {

			log.Println(fmt.Sprintf("listen and serve in port :%d", viper.GetInt("app.port")))
			if err := s.Serve(lis); err != nil {
				log.Println(fmt.Sprintf("failed to serve GRPC: %v", err))
			}

			cancel()

		}()

		go func() {

			<-ctx.Done()
			time.Sleep(5 * time.Second)

			s.GracefulStop()
			log.Println(fmt.Sprintf("grpc server is shutdown on port :%d", viper.GetInt("app.port")))

			if err := restSrv.Shutdown(ctx); err != nil {
				log.Fatalf("Server Shutdown Failed:%+v", err)
			}
			log.Println(fmt.Sprintf("rest server is shutdown on port :%d", viper.GetInt("app.port_rest")))

		}()

		log.Println(fmt.Sprintf("file host listen and serve in port :%d", viper.GetInt("app.port_rest")))
		if err := restSrv.ListenAndServe(); err != nil {
			log.Println(fmt.Sprintf("failed to serve Rest: %v", err))
		}

		// close any channel
		close(lobby.Lobby.Broadcast)

	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is github.com/renosyah/simpleUnoServer/.server.toml)")
	cobra.OnInitialize(initConfig)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func initConfig() {
	viper.SetConfigType("toml")
	if cfgFile != "" {

		viper.SetConfigFile(cfgFile)
	} else {

		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		viper.AddConfigPath(".")
		viper.AddConfigPath(home)
		viper.AddConfigPath("/etc/simpleUnoServer")
		viper.SetConfigName(".server")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func SignalContext(ctx context.Context) context.Context {

	ctx, cancel := context.WithCancel(ctx)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {

		<-sigs
		signal.Stop(sigs)
		close(sigs)
		cancel()
	}()
	return ctx
}
