#!/bin/bash

curl https://fastdl.mongodb.org/tools/db/mongodb-database-tools-ubuntu2004-x86_64-100.5.3.tgz -o mongodb-database-tools-ubuntu2004-x86_64-100.5.3.tgz
tar -zxvf mongodb-database-tools-*-100.5.3.tgz
mv mongodb-database-tools-*-100.5.3 /usr/local/bin/
echo 'export PATH=$PATH:/usr/local/bin/mongodb-database-tools-ubuntu2004-x86_64-100.5.3/bin' >> ~/.bashrc
rm mongodb-database-tools-*-100.5.3.tgz