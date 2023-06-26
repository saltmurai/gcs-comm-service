# Send a curl request to upload image

```
curl -X POST -F "image=@/path/to/image.jpg" http://localhost:5000/upload
```
Image must have the same ID with the mission received from GCS


# Mission file is saved in mission with the syntax: 

```
'{ID}-{Name}.json'
```
# Run comm service

```
docker compose up
```

Default is port 5000
Logging TCP port is on 5100
