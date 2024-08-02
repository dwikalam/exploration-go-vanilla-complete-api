package app

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"time"

	"github.com/dwikalam/ecommerce-service/internal/app/db/sqldb"
	"github.com/dwikalam/ecommerce-service/internal/app/handler/authhandler"
	"github.com/dwikalam/ecommerce-service/internal/app/handler/testhandler"
	"github.com/dwikalam/ecommerce-service/internal/app/helperdependency/config"
	"github.com/dwikalam/ecommerce-service/internal/app/helperdependency/crypto"
	"github.com/dwikalam/ecommerce-service/internal/app/helperdependency/logger"
	"github.com/dwikalam/ecommerce-service/internal/app/route"
	"github.com/dwikalam/ecommerce-service/internal/app/service/authsvc"
	"github.com/dwikalam/ecommerce-service/internal/app/service/testsvc"
	"github.com/dwikalam/ecommerce-service/internal/app/store/teststore"
	"github.com/dwikalam/ecommerce-service/internal/app/store/userstore"
	"github.com/dwikalam/ecommerce-service/internal/app/transaction"
	"golang.org/x/crypto/bcrypt"
)

func Run(
	ctx context.Context,
	stdout, stderr io.Writer,
) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	var (
		defaultLogger logger.Default
		cfg           config.EnvConfig
		psqlDB        sqldb.DB

		txManager transaction.SQLTransactionManager

		testStoreSQL teststore.SQLStore
		userStoreSQL userstore.SQLStore

		testService testsvc.Test
		authService authsvc.Auth

		testHandler testhandler.Test
		authHandler authhandler.Auth

		err error
	)

	defaultLogger = logger.NewDefault(stdout, stderr)

	cfg, err = config.NewEnvConfig()
	if err != nil {
		return fmt.Errorf("creating env config failed: %w", err)
	}

	bcrypt, err := crypto.NewBcrypt(bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("creating bcrypt failed: %w", err)
	}

	// Databases
	psqlDB, err = sqldb.NewDB(cfg.GetDbPsqlDriver(), cfg.GetDbPsqlDSN())
	if err != nil {
		return fmt.Errorf("creating psqldb failed: %w", err)
	}
	_, err = psqlDB.CheckHealth(ctx)
	if err != nil {
		return fmt.Errorf("check psqldb health failed: %w", err)
	}

	// Transaction Manager
	txManager, err = transaction.NewManager(&psqlDB)
	if err != nil {
		return fmt.Errorf("creating txManager failed: %w", err)
	}

	// Repositories
	testStoreSQL, err = teststore.NewTest(&psqlDB)
	if err != nil {
		return fmt.Errorf("creating testStoreSQL failed: %w", err)
	}
	userStoreSQL, err = userstore.NewSQLStore(&psqlDB)
	if err != nil {
		return fmt.Errorf("creating userStoreSQL failed: %w", err)
	}

	// Services
	testService, err = testsvc.NewTest(&txManager, &testStoreSQL)
	if err != nil {
		return fmt.Errorf("creating testService failed: %w", err)
	}
	authService, err = authsvc.NewAuth(&txManager, &userStoreSQL, &bcrypt)
	if err != nil {
		return fmt.Errorf("creating authService failed: %w", err)
	}

	// Handlers
	testHandler, err = testhandler.NewTest(&defaultLogger, &testService)
	if err != nil {
		return fmt.Errorf("creating testHandler failed: %w", err)
	}
	authHandler, err = authhandler.NewAuth(&defaultLogger, &authService)
	if err != nil {
		return fmt.Errorf("creating authHandler failed: %w", err)
	}

	srvMux := route.NewHttpHandler(
		&defaultLogger,
		&testHandler,
		&authHandler,
	)

	httpServer := &http.Server{
		Addr:         net.JoinHostPort(cfg.GetServerHost(), strconv.Itoa(cfg.GetServerPort())),
		Handler:      http.TimeoutHandler(srvMux, cfg.GetServerHandlerTimeout(), cfg.GetServerTimeoutMessage()),
		ReadTimeout:  cfg.GetServerReadTimeout(),
		WriteTimeout: cfg.GetServerWriteTimeout(),
		IdleTimeout:  cfg.GetServerIdleTimeout(),
	}

	var (
		wg sync.WaitGroup

		listenAndServe = func() {
			defaultLogger.Info(fmt.Sprintf("listening on %s", httpServer.Addr))

			err := httpServer.ListenAndServe()
			if err != nil && err != http.ErrServerClosed {
				defaultLogger.Error(fmt.Sprintf("error listening and serving %s: %s", httpServer.Addr, err))

				return
			}
		}

		shutdownGracefully = func(ctx context.Context, wg *sync.WaitGroup) {
			defer wg.Done()

			<-ctx.Done()

			shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()

			err := httpServer.Shutdown(shutdownCtx)
			if err != nil {
				defaultLogger.Error(fmt.Sprintf("shutting down http server: %s", err))
			}

			defaultLogger.Warn("server shutdown")

			err = psqlDB.Disconnect()
			if err != nil {
				defaultLogger.Error(fmt.Sprintf("closing database: %s", err))
			}

			defaultLogger.Warn("database closed")
		}
	)

	go listenAndServe()

	wg.Add(1)
	go shutdownGracefully(ctx, &wg)

	wg.Wait()

	return nil
}
