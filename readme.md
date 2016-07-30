# bilder - web app to host photo albums.

There's a demo available [here](https://geller.io/bilder/b/kitties).

**bilder** comes as a standalone webserver

## Setup

### bilder.json

This is the main configuration file for the bilder process. It currently supports the following options:

 + `url-path-prefix` *default:* `""`: This is a prefix that can be added to the assets' paths that are loaded from the browser. This allows running **bilder** behind a proxy like nginx that can terminate the HTTPS connection. Consider the path of the demo linked above: [https://geller.io/bilder/b/kitties](https://geller.io/bilder/b/kitties). In this case nginx proxy passes to the **bilder** process under the `/bilder` location:
```
location /bilder/ {
    proxy_pass http://localhost:8173/;
}
```
 + `bilder-dir` *default:* `"bilder"`: This is the path of the folder that **bilder** scans for album directories. For example, this directory would contain a single album `kitties` (please note that `index.html` and `*_thumb.jpg` are generated automatically by **bilder**):
```
$ find bilder
bilder
bilder/kitties
bilder/kitties/400.jpeg
bilder/kitties/400_thumb.jpeg
bilder/kitties/bilder.json
bilder/kitties/index.html
```
 + `access-log` *default:* `""`: When set to a file name, **bilder** logs requests against the `/b` path in combined log format to the set file. 

## Credits
**bilder** uses the following libraries and their dependencies:

 + @dimsemenov's [PhotoSwipe](https://github.com/dimsemenov/PhotoSwipe) for rendering the album.
 + @nfnt's [resize](https://github.com/nfnt/resize) to generate thumbnails.
 + @oliamb's [cutter](https://github.com/oliamb/cutter) to crop thumbnails to a centered square.
 + @satori's [go.uuid](https://github.com/satori/go.uuid) to generate a random session ID.
 + @gorilla's [handlers](https://github.com/gorilla/handlers) for logging requests.
 + 
