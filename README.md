# METAFILER documentation v1.0.8

## Doc Version

| Version | Date       | Author  | Description     |
| ------- | ---------- | ------- | --------------- |
| 1.0.0   | 17.06.2020 | mpetavy | Initial release |

## Description

METAFILER is provess to monitor a filesystem and index the file metadata to a MongoDB.

## Usage

* Start a mongodb instance as a docker container: docker run --rm -d -p 27017-27019:27017-27019 --name mongodb mongo:
  latest
* Connect to mongodb docker container: docker exec -it mongodb bash

## License

All software is copyright and protected by the Apache License, Version 2.0.
https://www.apache.org/licenses/LICENSE-2.0