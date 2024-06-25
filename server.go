package main

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

// Estrutura para representar um Tile
type Tile struct {
	X    int
	Y    int
	Zoom int
}

type Envelope struct {
	XMin float64
	XMax float64
	YMin float64
	YMax float64
}

func tileToEnvelope(tile Tile) Envelope {
	worldMercMax := 20037508.3427892
	worldMercMin := -worldMercMax
	worldMercSize := worldMercMax - worldMercMin
	worldTileSize := math.Pow(2, float64(tile.Zoom))
	tileMercSize := worldMercSize / worldTileSize

	env := Envelope{
		XMin: worldMercMin + tileMercSize*float64(tile.X),
		XMax: worldMercMin + tileMercSize*float64(tile.X+1),
		YMin: worldMercMax - tileMercSize*float64(tile.Y+1),
		YMax: worldMercMax - tileMercSize*float64(tile.Y),
	}

	return env
}

func envelopeToBoundsSQL(env Envelope) string {
	densifyFactor := 4
	segSize := (env.XMax - env.XMin) / float64(densifyFactor)
	sqlTmpl := fmt.Sprintf("ST_Segmentize(ST_MakeEnvelope(%f, %f, %f, %f, 3857), %f)",
		env.XMin, env.YMin, env.XMax, env.YMax, segSize)
	return sqlTmpl
}

func envelopeToSQL(env Envelope, pk int) string {
	envSQL := envelopeToBoundsSQL(env)
	geomColumn := "geom"
	table := "maps_layers_geometries"
	layer := pk

	sqlTmpl := fmt.Sprintf(`
        WITH 
        bounds AS (
            SELECT %s AS geom, 
                   %s::box2d AS b2d
        ),
        mvtgeom AS (
            SELECT ST_AsMVTGeom(ST_Transform(t.%s, 3857), bounds.b2d) AS geom,
			t.id_geometry,
			t.layer_id as id_layer
            FROM %s t, bounds
            WHERE ST_Intersects(t.%s, ST_Transform(bounds.geom, 4326))
            AND layer_id = %d
        ) 
        SELECT ST_AsMVT(mvtgeom.*) FROM mvtgeom
    `, envSQL, envSQL, geomColumn, table, geomColumn, layer)

	return sqlTmpl
}

func queryVectorTile(db *sql.DB, sqlQuery string) ([]byte, error) {
	var result []byte
	err := db.QueryRow(sqlQuery).Scan(&result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func getVectorTile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	z, err := strconv.Atoi(vars["z"])
	if err != nil {
		http.Error(w, "Invalid zoom level", http.StatusBadRequest)
		return
	}
	x, err := strconv.Atoi(vars["x"])
	if err != nil {
		http.Error(w, "Invalid x coordinate", http.StatusBadRequest)
		return
	}
	y, err := strconv.Atoi(vars["y"])
	if err != nil {
		http.Error(w, "Invalid y coordinate", http.StatusBadRequest)
		return
	}
	pk, err := strconv.Atoi(vars["pk"])
	if err != nil {
		http.Error(w, "Invalid layer ID", http.StatusBadRequest)
		return
	}

	tile := Tile{
		Zoom: z,
		X:    x,
		Y:    y,
	}
	env := tileToEnvelope(tile)
	sqlQuery := envelopeToSQL(env, pk)

	db, err := sql.Open("postgres", "host=postgis port=5432 user=postgres password=geoadmin dbname=postgres sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	pbf, err := queryVectorTile(db, sqlQuery)
	if err != nil {
		http.Error(w, "Database query failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/vnd.mapbox-vector-tile")
	w.Write(pbf)
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/layers/{pk:[0-9]+}/vectortiles/{z:[0-9]+}/{x:[0-9]+}/{y:[0-9]+}.pbf/", getVectorTile).Methods("GET")

	// Adiciona o middleware CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"}, // Permite todas as origens
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	handler := c.Handler(r)
	log.Fatal(http.ListenAndServe(":8080", handler))
}
