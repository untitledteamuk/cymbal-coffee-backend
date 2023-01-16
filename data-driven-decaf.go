package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"cloud.google.com/go/alloydbconn"
	"cloud.google.com/go/alloydbconn/driver/pgxv4"
	"cloud.google.com/go/cloudsqlconn"
	"cloud.google.com/go/cloudsqlconn/mysql/mysql"
	"github.com/go-chi/chi"
	"github.com/jackc/pgx/v4/pgxpool"
)

type DDDBondPayload struct {
	MagicCoffee string `json:"magic_coffee,omitempty"`
	Total       int    `json:"total,omitempty"`
	Project     string `json:"project,omitempty"`
	DB          string `json:"db,omitempty"`
}

type DBConnectionInfo struct {
	User       string
	Pass       string
	DBName     string
	DBRegion   string
	DBCluster  string
	DBInstance string
	ProjectID  string
}

func dbConnectionInfo() (info DBConnectionInfo, err error) {
	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASS")
	dbName := os.Getenv("DB_NAME")
	dbRegion := os.Getenv("DB_REGION")
	dbCluster := os.Getenv("DB_CLUSTER")
	dbInstance := os.Getenv("DB_INSTANCE")
	dbProject := os.Getenv("DB_PROJECT")
	if user == "" || pass == "" || dbInstance == "" || dbName == "" {
		return info, fmt.Errorf("ensure required environment variables are set")
	}
	if dbProject == "" {
		dbProject = cfg.ProjectID
	}
	info.User = user
	info.Pass = pass
	info.DBName = dbName
	info.DBRegion = dbRegion
	info.DBCluster = dbCluster
	info.DBInstance = dbInstance
	info.ProjectID = dbProject
	return info, nil
}

const defaultQuery = "select * from coffee"

// Init AlloyDB and MySQL driver registration on startup
func DDDInit() error {
	alloyDBCleanup, err := pgxv4.RegisterDriver("alloydb")
	if err != nil {
		log.Printf("failed to parse pgx config: %v\n", err)
		return err
	}
	defer alloyDBCleanup()

	mySQLCleanup, err := mysql.RegisterDriver("cloudsql-mysql")
	if err != nil {
		log.Printf("failed to parse pgx config: %v\n", err)
		return err
	}
	defer mySQLCleanup()

	return nil
}

func DDDMySQLConnect(ctx context.Context) (result DDDBondPayload, err error) {

	info, err := dbConnectionInfo()
	if err != nil {
		log.Printf("Error: Cannot load database info: %v\n", err)
		return result, err
	}
	dsn := fmt.Sprintf("%s:%s@cloudsql-mysql(%s:%s:%s)/%s", info.User, info.Pass, info.ProjectID, info.DBRegion, info.DBInstance, info.DBName)

	db, err := sql.Open(
		"cloudsql-mysql",
		dsn)
	if err != nil {
		log.Printf("failed to connect: %v\n", err)
		return result, err
	}
	rows, err := db.Query(defaultQuery)
	if err != nil {
		log.Printf("query failed: %v\n", err)
		return result, err
	}

	defer rows.Close()

	var (
		i     int
		bean  string
		price string
	)
	for rows.Next() {
		err = rows.Scan(&i, &bean, &price)
		if err != nil {
			log.Printf("query failed: %v\n", err)
			return result, err
		}
		if i == 51 {
			result.MagicCoffee = bean
		}
		p, err := strconv.Atoi(strings.Split(price, ".")[0])
		if err != nil {
			log.Printf("Could not convert %v to an integer\n", price)
			continue
		}
		result.Total += p
	}
	return result, nil
}

// Create a postgres connection (same for AlloyDB and CloudSQL)
func DDDPostgresConnection() (c *pgxpool.Config, err error) {
	info, err := dbConnectionInfo()
	if err != nil {
		log.Printf("Error: Cannot load database info: %v\n", err)
		return c, err
	}
	dsn := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", info.User, info.Pass, info.DBName)
	c, err = pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Printf("failed to parse pgx config: %v\n", err)
		return c, err
	}
	return c, nil
}

// Connect to AlloyDB
func DDDAlloyConnect(ctx context.Context) (result DDDBondPayload, err error) {
	c, err := DDDPostgresConnection()
	if err != nil {
		log.Printf("failed to parse pgx config: %v\n", err)
		return result, err
	}
	// Create a new dialer with any options
	d, err := alloydbconn.NewDialer(ctx)
	if err != nil {
		log.Printf("failed to initialize dialer: %v\n", err)
		return result, err
	}
	defer d.Close()

	info, err := dbConnectionInfo()
	if err != nil {
		log.Printf("Error: Cannot load database info: %v\n", err)
		return result, err
	}
	if info.DBCluster == "" {
		log.Printf("Error: DB_CLUSTER not set (required for alloydb): %v\n", err)
		return result, fmt.Errorf("expected db cluster to be set")
	}

	// Tell the driver to use the Cloud SQL Go Connector to create connections
	c.ConnConfig.DialFunc = func(ctx context.Context, _ string, instance string) (net.Conn, error) {
		return d.Dial(ctx, fmt.Sprintf("projects/%s/locations/%s/clusters/%s/instances/%s", info.ProjectID, info.DBRegion, info.DBCluster, info.DBInstance))
	}

	// Interact with the driver directly as you normally would
	pool, err := pgxpool.ConnectConfig(context.Background(), c)
	if err != nil {
		log.Printf("failed to connect: %v\n", err)
		return result, err
	}
	defer pool.Close()
	// Consistent for AlloyDB and Postgres
	return DDDPostgresRows(ctx, pool)
}

// Connect to CloudSQL Postgres
func DDDPostgresConnect(ctx context.Context) (result DDDBondPayload, err error) {
	c, err := DDDPostgresConnection()

	// Create a new dialer with any options
	d, err := cloudsqlconn.NewDialer(context.Background())
	if err != nil {
		log.Printf("failed to initialize dialer: %v\n", err)
		return result, err
	}
	defer d.Close()
	info, err := dbConnectionInfo()
	if err != nil {
		log.Printf("Error: Cannot load database info: %v\n", err)
		return result, err
	}
	// Tell the driver to use the Cloud SQL Go Connector to create connections
	c.ConnConfig.DialFunc = func(ctx context.Context, _ string, instance string) (net.Conn, error) {
		return d.Dial(ctx, fmt.Sprintf("%s:%s:%s", info.ProjectID, info.DBRegion, info.DBInstance))
	}

	// Interact with the driver directly as you normally would
	pool, err := pgxpool.ConnectConfig(context.Background(), c)
	if err != nil {
		log.Printf("failed to connect: %v\n", err)
		return result, err
	}
	defer pool.Close()
	// Consistent for AlloyDB and Postgres
	return DDDPostgresRows(ctx, pool)
}

// Process Pogres rows (same for alloydb and cloud sql)
func DDDPostgresRows(ctx context.Context, pool *pgxpool.Pool) (result DDDBondPayload, err error) {
	rows, err := pool.Query(ctx, defaultQuery)
	if err != nil {
		log.Printf("query failed: %v\n", err)
		return result, err
	}
	defer rows.Close()

	i := 0
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			log.Printf("query failed: %v\n", err)
			return result, err
		}
		if i == 50 {
			result.MagicCoffee = values[1].(string)
		}
		price := values[2].(string)
		p, err := strconv.Atoi(strings.Split(price, ".")[0])
		if err != nil {
			log.Printf("Could not convert %v to an integer\n", price)
			continue
		}
		result.Total += p
		i++
	}
	return result, nil
}

// Chi router to handle incoming GET
func dddRouter(r chi.Router) {
	r.Get("/", dddHandler)
	//r.Post("/cloud_sql_postgres", eventHandler)
	//r.Post("/cloud_sql_mysql", eventHandler)
}

func dddHandler(w http.ResponseWriter, r *http.Request) {

	var result DDDBondPayload
	var err error

	switch os.Getenv("DB_TYPE") {
	case "ALLOY_DB":
		result, err = DDDAlloyConnect(r.Context())
		if err != nil {
			log.Printf("Data-Driven Decaf: Error: %v\n", err)
			http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
			return
		}
	case "CLOUD_SQL_POSTGRES":
		result, err = DDDPostgresConnect(r.Context())
		if err != nil {
			log.Printf("Data-Driven Decaf: Error: %v\n", err)
			http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
			return
		}
	case "CLOUD_SQL_MYSQL":
		result, err = DDDMySQLConnect(r.Context())
		if err != nil {
			log.Printf("Data-Driven Decaf: Error: %v\n", err)
			http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
			return
		}
	default:
		// Don't know the DB type, error out
		log.Printf("Data-Driven Decaf: Unknown DB type %v\n", os.Getenv("DB_TYPE"))
		http.Error(w, fmt.Sprintf("Error: Unknown DB type %v", os.Getenv("DB_TYPE")), http.StatusInternalServerError)
		return
	}
	// Add Project ID and DB type to results
	result.Project = cfg.ProjectID
	result.DB = os.Getenv("DB_TYPE")

	log.Printf("Result: %+v", result)

	// Verify with Bond Service
	res, err := sendJson(r.Context(), "/v1/data_driven_decaf/verify", result)
	if err != nil {
		if res != nil {
			log.Printf("Data-Driven Decaf: Error: Body: %v", string(res))
		}
		log.Printf("Data-Driven Decaf: Error: %v\n", err)
		http.Error(w, fmt.Sprintf("Data-Driven Decaf Error: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Response: %v\n", res)

	json.NewEncoder(w).Encode(result)

}
