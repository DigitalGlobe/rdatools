package main

import (
	"log"
	"os/user"
	"path/filepath"

	"net/http"

	"time"

	"encoding/json"
	"fmt"

	"github.com/BurntSushi/toml"
	"golang.org/x/oauth2"
)

type Config struct {
	Username string        `mapstructure:"gbdx_username" toml:"gbdx_username"`
	Password string        `mapstructure:"gbdx_password" toml:"gbdx_password"`
	Token    *oauth2.Token `mapstructure:"gbdx_token" toml:"gbdx_token"`
}

type Metadata struct {
	ImageMetadata struct {
		NumXTiles   int
		NumYTiles   int
		TileXSize   int
		TileYSize   int
		ImageWidth  int
		ImageHeight int
		NumBands    int
		MinX        int
		MinY        int
		MinTileX    int
		MinTileY    int
		MaxTileX    int
		MaxTileY    int
		DataType    string
	}
	ImageGeoreferencing struct {
		SpatialReferenceSystemCode string
		ScaleX                     float64
		ScaleY                     float64
		TranslateX                 float64
		TranslateY                 float64
		ShearX                     float64
		ShearY                     float64
	}
}

func main() {

	// graphID := "2266e5a362b71333c95d59ecd25f6bac1d8954779f9bd1629b904f8a542a88cd"
	// nodeID := "BandSelect-RGB"
	//graphID := "be3380f89ac8d4a5eef4d78549f183284d61f0fccbdfed6c17a1c36ac6b38d92"
	//nodeID := "FormatByte"
	graphID := "d51ac576f2938fdbd09047f5b2d1fe0fbf7da345846b2861c4dc72038a15bb36"
	nodeID := "SmartBandSelect_blmsvy"

	log.SetFlags(log.Lshortfile)

	// Get GBDX creds.
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	credPath := filepath.Join(usr.HomeDir, ".gbdx", "credentials.toml")
	confMap := make(map[string]Config)
	if _, err := toml.DecodeFile(credPath, &confMap); err != nil {
		log.Fatalln("failed decoding credentials", err)
	}
	conf, ok := confMap["default"]
	if !ok {
		log.Fatalln("no default gbdx credentials found to use")
	}

	// Get Metadata.
	client := &http.Client{
		Timeout: 60 * time.Second,
	}
	req, err := http.NewRequest("GET", fmt.Sprintf("https://rda.geobigdata.io/v1/metadata/%s/%s/metadata.json", graphID, nodeID), nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", conf.Token.AccessToken))
	res, err := client.Do(req)
	if err != nil {
		log.Fatalln(req, err)
	}
	defer res.Body.Close()

	md := Metadata{}
	json.NewDecoder(res.Body).Decode(&md)

	fmt.Print(md)

	// // Download tiles.
	// r, err := NewRetriever(graphID, nodeID, md, "tmpPS", conf.Token)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// ts := time.Now()
	// tileMap := r.Retrieve()
	// elapsed := time.Since(ts)
	// log.Printf("Tile retrieval took %s", elapsed)

	// // Build VRT.
	// log.Println(len(tileMap))
	// vrt, err := NewVRT(&md, tileMap)
	// if err != nil {
	// 	log.Fatal("failed building vrt")
	// }

	// f, err := os.Create("rdaPS.vrt")
	// if err != nil {
	// 	log.Fatal("failure creating VRT file")
	// }
	// defer f.Close()

	// enc := xml.NewEncoder(f)
	// enc.Indent("  ", "    ")
	// if err := enc.Encode(vrt); err != nil {
	// 	fmt.Printf("error: %v\n", err)
	// }
}
