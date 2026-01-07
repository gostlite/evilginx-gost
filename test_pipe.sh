#!/bin/bash
echo "Sending input to batch..."
(echo "LINE_ONE"; echo "LINE_TWO"; sleep 1) | cmd //c check_pipe.bat > pipe_test.log 2>&1
echo "Done."
cat pipe_test.log
