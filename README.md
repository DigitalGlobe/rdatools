# rdatools

A cli for accessing RDA resources.

## `rda`

The `rda` command line tool is a Go based executable for accessing RDA functionality.  As such, it is a statically linked executable and should run without hassle on most systems.

## Using `rda`

In general, `rda --help` is your guide.  `--help` works for all subcommands as well.  Some of the commands return JSON responses; formatting JSON is easy by piping the output of the command to `jq` or `python -m json.tool`.  For instance, `rda operator DigitalGlobeStrip1B | jq` yields nicely formatted JSON describing that operator.

### `rda configure`

The first time you use `rda`, you need to configure it to store your GBDX credentials (or set the environment variables `GBDX_USERNAME` and `GBDX_PASSWORD`).  Once configured, `rda` will cache your GBDX token and refresh it on demand for you without you needing to intervene.  Simply type
```
rda configure
```
and provide the requested GBDX credentials.  Note that it is possible to provide a `--profile` if you have more than one set of GBDX credentials (this is similar to how the AWS cli behaves).  If `--profile` is not provided, _default_ is used.  All subcommands support `--profile`.

### `rda token`

This will return to you a valid GBDX token.  If the cached one is not set or expired, it will be refreshed before returned to you.  This is nice if you want to use `curl` or postman and need a token ASAP.

### `rda operator`

Returns JSON describing all the RDA operators available.  To get information on a single operator, you just specify the name, e.g. `rda operator DigitalGlobeStrip`. JSON is returned so you may want to pipe the output to a formatting tool.

### `rda dgstrip`

`rda dgstrip` will concurrently download and create a VRT composed of tiles _realized_ from RDA.  Note that you can provide either `--srcwin` or `--projwin` flags if you don't want to download an entire image, which is usually a good idea.  `rda dgstrip` downloads all tiles that intersect the provided window where that window intersects the global image window.

Try out `rda dgstrip --help` to see the flags you can provide to control how the image is processed.

For example,
```
rda dgstrip 103001000EBC3C00 103001000EBC3C00-ovr.vrt --gsd 0.000146 --dra --bands RGB --bandtype PS --crs EPSG:4326 --projwin -116.79,37.86,-116.70,37.78
```
Will return a downsampled version of catalog id `103001000EBC3C00` to you as a VRT.  Just load it up into QGIS and profit! 

The actual tiles are stored in a directory named `103001000EBC3C00` adjacent to the VRT.  The VRT format is an xml based format that describes how to lay out the tiles as if they were a single image.  You can create a single geotiff out of the downloaded product via GDAL, e.g. `gdal_translate 103001000EBC3C00.vrt 103001000EBC3C00.tif` should do it if you have GDAL installed.

### `rda dgstrip metadata` 

`rda dgstrip metadata` will return JSON describing what you can download via `rda dgstrip`.  Try it out via
```
rda dgstrip 103001000EBC3C00 --gsd 0.000146 --dra --bands RGB --bandtype PS --crs EPSG:4326
```
and you'll get a response desribing how large `103001000EBC3C00` is in a geographic projection (EPSG:4326 is the code for a lat/long projection; you can specify any that you like via the `--crs` flag).

### `rda stripinfo`

`rda stripinfo` returns JSON desribing a collection.  For instance, `rda stripinfo 103001000EBC3C00` will give you all the information you might want about `103001000EBC3C00`.  Remember to pipe this to a JSON formatter if you want a pretty view of it.
