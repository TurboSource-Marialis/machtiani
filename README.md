# machtiani

Code chat, against retrieved files from commits.

## Quick launch

Clone this project.

Add add, fetch, and load a repo to be indexed - see the [commit-file-retrieval readme](machtiani-commit-file-retrieval/README.md).

Launch.

```
docker-compose up --build
```

## Usage

After launch, try machtiani's only endpoint [generate-response](http://localhost:5071/docs#/default/generate_response_generate_response_post).

## Todo

- [ x ] Retrieve file content and add to prompt.

- [ x ] Fetch on UI is tempermental, if wrong url and token given, it will messup. Maybe all that should be done strictly on commit-file-retrieval server side, the url and token. just pass project name

- [ x ] Separate command for sending edited markdown (don't wrap # User)
        (completed with commmit 5a69231d4b48b6cd8c1b1e3b54a1b57c3d295a74)

