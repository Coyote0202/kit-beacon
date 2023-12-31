# SPDX-License-Identifier: MIT
#
# Copyright (c) 2023 Berachain Foundation
#
# Permission is hereby granted, free of charge, to any person
# obtaining a copy of this software and associated documentation
# files (the "Software"), to deal in the Software without
# restriction, including without limitation the rights to use,
# copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the
# Software is furnished to do so, subject to the following
# conditions:
#
# The above copyright notice and this permission notice shall be
# included in all copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
# EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES
# OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
# NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
# HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
# WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
# FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
# OTHER DEALINGS IN THE SOFTWARE.

if [ -z "$CHAINID" ]; then
    CHAINID="brickchain-666"
fi
if [ -z "$KEYRING" ]; then
    KEYRING="test"
fi
if [ -z "$KEYALGO" ]; then
    KEYALGO="secp256k1"
fi
if [ -z "$LOGLEVEL" ]; then
    LOGLEVEL="info"
fi
if [ -z "$HOMEDIR" ]; then
    HOMEDIR="/root/.beacond"
fi

KEY="$1"
MONIKER="$1"
TRACE=""
GENESIS=$HOMEDIR/config/genesis.json
TMP_GENESIS=$HOMEDIR/config/tmp_genesis.json


beacond init $MONIKER -o --chain-id $CHAINID --home "$HOMEDIR"

beacond config set client keyring-backend $KEYRING --home "$HOMEDIR"

beacond keys add $KEY --keyring-backend $KEYRING --algo $KEYALGO --home "$HOMEDIR"
