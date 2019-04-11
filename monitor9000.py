import serial
import serial.tools.list_ports
import time
import psutil

ports = [x.device for x in serial.tools.list_ports.comports(include_links=False)]
print ports

arduino = serial.Serial(ports[-1], 9600) # timeout=2
time.sleep(5)
print arduino

def send(pin, state):
    arduino.write('%s\n' % pin)
    time.sleep(0.1)
    arduino.write('%s\n' % state)   
    time.sleep(0.1)
    print 'Sent: %s %s' % (pin, state)

send('1', '255') # turn the lights on

cpu = psutil.cpu_percent(interval=None)

while True:
    cpu = psutil.cpu_percent(interval=None)
    # print cpu
    cpu = max(min(int(cpu * 2.55), 255), 0)

    send('0', str(cpu))
    time.sleep(0.5)
