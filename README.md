# monitor9000 

![RECLibboard](img/monitor.jpg)

This makes ancient Soviet ammeter display CPU load using Arduino. It can potentially display anything and have some kind of GUI but I didn't bother to implement it yet. 

## Installing
```
pip install -r requirements.txt
```

It can work as a windows service
```
go build 
sc create Monitor9000 binpath= "D:\projects\git\monitor9000\monitor9000.exe"
```