kill -9 $(lsof -i:35601 |awk '{print $2}' | tail -n 1)
nohup ./server/sergate --config=server/config.json --web-dir=$(dirname $0 | pwd)/web/ >> run.log 2>&1 &
cd ..
