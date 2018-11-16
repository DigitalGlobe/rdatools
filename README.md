# rdatools

A cli for accessing RDA resources.

## `rda`

The `rda` command line tool is a Go based executable for accessing RDA functionality.  As such, it is a statically linked executable and should run without hassle on most systems.

# Installation

To install `rda`, navigate to releases page [here](https://github.com/DigitalGlobe/rdatools/releases)  and download the most recent package for your operating system (note that Darwin is Max OSX).  Unpack your download and you will find a binary executable named `rda`.  Place this in your path so that you can access it from the command line wherever you're at, or run it directly from where you downloaded it.

## Using `rda`

In general, `rda --help` is your guide, and note that `--help` works for all subcommands as well.  You can find what version of `rda` you're running via `rda --version`.

Some of the commands return JSON responses; formatting JSON is easy by piping the output of the command to `jq` or `python -m json.tool`.  For instance, `rda operator DigitalGlobeStrip1B | jq` yields nicely formatted JSON describing that operator.

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

### `rda stripinfo`

`rda stripinfo` returns JSON desribing the given catalog.  For instance, `rda stripinfo 103001000EBC3C00` will give you all the information you might want about `103001000EBC3C00`.  Remember to pipe this to a JSON formatter if you want a pretty view of it.

In addition, you can provide a `--zipfile <location of zip>` (IMD files, etc) to download the original metadata that came with the imagery when provided by DG's internal factory.

### `rda dgstrip`

`rda dgstrip` is a subcommand offering the additional commands `metadata`, `realize`, and `batch`, described below.

#### `rda dgstrip metadata`

`rda dgstrip metadata` will return JSON describing what you can download via `rda dgstrip realize` or `rda dgstrip batch`.  Try it out via
```
rda dgstrip metadata 103001000EBC3C00 --gsd 0.000146 --dra --bands RGB --bandtype PS --crs EPSG:4326
```
and you'll get a response desribing how large `103001000EBC3C00` is in a geographic projection (EPSG:4326 is the code for a lat/long projection; you can specify any that you like via the `--crs` flag; the default projection is UTM in the zone that strip is located).

#### `rda dgstrip realize`

This command concurrently downloads and creates a VRT composed of tiles _realized_ from RDA.  Note that you can provide either `--srcwin` or `--projwin` flags if you don't want to download an entire image, which is usually a good idea, as large images are best aquired using `rda dgstrip batch`.  `rda dgstrip realize` downloads all tiles that intersect the provided window where that window intersects the global image window.

Try out `rda dgstrip realize --help` to see the flags you can provide to control how the image is processed.

For example,
```
rda dgstrip realize 103001000EBC3C00 103001000EBC3C00-ovr.vrt --gsd 0.000146 --dra --bands RGB --bandtype PS --crs EPSG:4326 --projwin -116.79,37.86,-116.70,37.78
```
Will return a downsampled version of catalog id `103001000EBC3C00` to you as a VRT.  Just load it up into QGIS/ArcGIS/your favorite viewer that can read VRTs and profit! 

The actual tiles are stored in a directory named `103001000EBC3C00` adjacent to the VRT.  The VRT format is an xml based format that describes how to lay out the tiles as if they were a single image.  You can create a single geotiff out of the downloaded product via GDAL, e.g. `gdal_translate 103001000EBC3C00.vrt 103001000EBC3C00.tif` should do it if you have GDAL installed.

#### `rda dgstrip batch` 

`batch` takes all the same flags as `realize` (except the vrt location), but rather than realize the tiles it submits a batch materialization request to RDA.  You will get a response that includes a job id, which as you'll see below you can use to status and download the output of the batch materialization job. For example, running
```
rda dgstrip batch 103001000EBC3C00 103001000EBC3C00-ovr.vrt --gsd 0.000146 --dra --bands RGB --bandtype PS --crs EPSG:4326 --projwin -116.79,37.86,-116.70,37.78
```
The output of this is a json message, that includes a field "jobId" whos values you can use as described below.

### `rda job`

`rda job` hosts subcommands lets you status and download the outputs from RDA's batch materialization endpoint. The subcommands of interest are `download`, `downloadable`, `status`, and `watch`.

#### `rda job downloadable`

This command returns all the job ids that are listed in your GBDX customer data bucket under the rda prefix.  You should be able to `status`, `download`, or `watch` these jobs.

You can optionally provide a job id as an argument; if you do so, you will be returned a list of the objects that can be downloaded for that job id.

#### `rda job download`

`download` will download all the outputs for the given job id that are in S3.  Use the `watch` subcommand to watch a job and greedily download outputs as they arrive.

Note that you can also provide the path to an individual object (e.g. a path returned from `rda job downloadable` where you provide a job id as an argument) to pull down just that object.  This is implemented via prefix matching, so in reality you can provide a prefix to match as the job id and all matching objects will be returned.  This is similar to how the aws cli command functions.

#### `rda job status`

This returns the status of the given job id associated with an RDA materialization request.

#### `rda job watch`

This is a combination of the functionality of `download` and `status`; essentially it polls RDA for job status, greedily downloading any of its produced outputs as they are created, and continues to poll until the job is complete and all outputs are downloaded to the given output directory.

For example, running
```
rda job watch ~/Downloads/rdaout 21a12531-2bfe-4e29-84b0-52b9433f7a61
```
downloads the output of job id `21a12531-2bfe-4e29-84b0-52b9433f7a61` to `~/Downloads/rdaout` on my machine.

#### `rda job rm`

This removes all artifacts in S3 associated with the a given RDA batch job id.  For instance, 
```
rda job rm 21a12531-2bfe-4e29-84b0-52b9433f7a61
```
would remove all S3 objects in your GBDX customer data bucket associated with the batch job `21a12531-2bfe-4e29-84b0-52b9433f7a61`.

