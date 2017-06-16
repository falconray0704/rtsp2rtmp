# rtsp2rtmp
rtsp stream to rtmp

How to use:

cd demo
go build demo.go
./demo -rtspUrl "rtsp://admin:123456@10.1.51.13/H264?ch=1&subtype=0" -rtmpUrl "rtmp://10.1.51.20:1935/myapp/cv"

