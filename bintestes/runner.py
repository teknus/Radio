import os, sys

while True:
    try:
        param = "./client_listenner " + sys.argv[1]
        os.system("{} | mpg123 -".format(param))
    except KeyboardInterrupt:
        sys.exit(0)
