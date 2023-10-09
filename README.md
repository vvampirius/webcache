# webcache
Simple Web Cache Server

It stores remote web items, so you may use them then without worrying about original availability.

Just use URL [http://your_webcache_instance?url=https://example.com/example.jpg](http://your_webcache_instance/?url=https%3A%2F%2Fexample.com%2Fexample.jpg) instead https://example.com/example.jpg .

## Run

```shell
./webcache -i :8080 -p passwords.json path_to_store
```

passwords.json:
```json
{"my_username": "my_password"}
```

But I recommend protecting your instance with Nginx with SSL and some request size limiting.

## Usage

If you use a browser and webcache runs with passwords file: Firstly, visit to [main page](http://localhost:8080) to authorize.

Store and get remote item from webcache:

```shell
curl -u my_username:my_password 'http://localhost:8080?url=https://example.com/example.jpg'
```

Get item from webcache if it exists (and webcache runs with passwords file):

```shell
curl 'http://localhost:8080?url=https://example.com/example.jpg'
```

Store remote item for 10 hours and get it from webcache:

```shell
curl -u my_username:my_password 'http://localhost:8080?url=https://example.com/example.jpg&ttl=10h'
```

Store large remote item in background:

```shell
curl -u my_username:my_password 'http://localhost:8080?url=https://example.com/big_example.jpg&background'
```

Delete item from webcache:
```shell
curl -u my_username:my_password -X DELETE 'http://localhost:8080?url=https://example.com/example.jpg'
```

Put local item to webcache:
```shell
curl -u my_username:my_password -T local_example.jpg 'http://localhost:8080?url=https://example.com/example.jpg'
```