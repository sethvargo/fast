# Fast CLI
Fast CLI is a small tool which leverages Netflix's [fast.com](https://fast.com).

## Usage
```shell
fast
Getting API token...
Starting download test...

Speed: 170.98 Mbps
```

## Installation
1. Download from GitHub releases:

    ```shell
    curl -sLo fast.tgz https://github.com/sethvargo/fast/releases/download/v0.1.1/fast_0.1.1_darwin_amd64.tgz
    ```

1. Untar:

    ```shell
    tar -zxvf fast.tgz
    ```

1. Move into bin or $PATH:

    ```shell
    chmod +x fast
    mv fast /usr/local/bin/
    ```

1. Test

    ```shell
    fast
    ```

## TODO
- [ ] Use concurrent downloads if the client is fast enough
- [ ] Better metrics aggregation (ignore outliers)

## Disclaimer

This project is not associated with Netflix in any way. Do not do bad things.
Thanks.
