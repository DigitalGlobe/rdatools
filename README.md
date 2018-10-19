# rdatools

A cli for accessing RDA resources.

## `rda`

The `rda` command line tool is a Go based executable for accessing RDA functionality.  As such, it is a statically linked executable and should run without hassle on most systems.

## Using `rda`

In general, `rda --help` is your guide.  `--help` works for all subcommands as well.

### `rda configure`

The first time you use `rda`, you need to configure it to store your GBDX credentials (or set the environment variables `GBDX_USERNAME` and `GBDX_PASSWORD`).  Once configured, `rda` will cache your GBDX token and refresh it on demand for you without you needing to intervene.  Simply type
```
rda configure
```
and provide requested information.  Note that it is possible to provide a `--profile` if you have more than one set of credentials (this is similar to how the AWS cli behaves).  If `--profile` is not provided, _default_ is used.  All subcommands support `--profile`.

### `rda token`

This will return to you a valid GBDX token.  If the cached one is not set or expired, it will be refreshed before returned to you.

### `rda metadata`

Given an RDA graph and a node id contained in the graph, this will return the metadata that describes the imagery you can fetch from that graph.

### `rda realize`

`rda realize` will concurrently download and create a VRT composed of tiles _realized_ from an RDA graph and node.  Note that you can provide either `--srcwin` or `--projwin` flags if you don't want to download an entire image, which is usually a good idea.  `rda realize` downloads all tiles that intersect the provided window where that window intersects the global image window.




