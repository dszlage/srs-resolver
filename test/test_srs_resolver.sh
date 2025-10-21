#!/bin/bash

# Test script for srs-resolver
# Usage: ./test_srs_resolver.sh
# Make sure srs-resolver is running on 127.0.0.1:10022

HOST="127.0.0.1"
PORT="10022"
TIMEOUT="2"

echo "=== SRS Resolver Test Script ==="
echo "Testing connection to $HOST:$PORT"
echo

# Function to test a single address
test_address() {
    local address="$1"
    local description="$2"
    
    echo -n "Testing: $description"
    echo -n " [$address] -> "
    
    result=$(printf "get %s\n" "$address" | timeout $TIMEOUT nc $HOST $PORT 2>/dev/null)
    
    if [ $? -eq 0 ] && [ -n "$result" ]; then
        echo "$result"
    else
        echo "ERROR: No response or connection failed"
    fi
}

echo "1. VALID EMAIL ADDRESSES (common cases):"
test_address "user@example.com" "Standard email"
test_address "user@domain.pl" "Email with Polish domain"
test_address "test.email@subdomain.example.org" "Email with subdomain"
test_address "admin+tag@company.com" "Email with tag"
echo

echo "2. VALID SRS0 ADDRESSES:"
test_address "SRS0=abc123=timestamp=originaldomain.com=localpart" "SRS0 basic"
test_address "SRS0=hash123=12345=example.com=damian" "SRS0 with timestamp"
test_address "SRS0=xyz789=67890=domain.pl=admin" "SRS0 with Polish domain"
echo

echo "3. VALID SRS1 ADDRESSES:"
test_address "SRS1=def456=timestamp=originaldomain.com=localpart" "SRS1 basic"
test_address "SRS1=hash456=12345=example.com=user.name" "SRS1 with dot in name"
echo

echo "4. SRS WITH FULL ADDRESS IN LOCAL PART:"
test_address "SRS0=abc123=12345=domain.pl=user.surname@domain.tdl" "SRS0 with full address"
test_address "SRS1=def456=67890=example.com=user@forwarder.com" "SRS1 with full address"
echo

echo "5. INVALID FORMATS (testing fallback):"
test_address "invalid-email-without-at" "Email without @"
test_address "email@" "Email with empty domain"
test_address "@domain.com" "Email with empty local part"
test_address "email with spaces@domain.com" "Email with spaces"
test_address "email<with>forbidden@chars.com" "Email with forbidden characters"
test_address "SRS0=incomplete" "Incomplete SRS0"
test_address "SRS0=too=few=parts" "SRS0 with too few parts"
test_address "SRS0=too=many=equal=signs=here=and=more" "SRS0 with too many equal signs"
test_address "completely-invalid-input" "Completely invalid input"
echo

echo "6. INVALID PROTOCOL COMMANDS:"
test_address "" "Empty command (only get)"
printf "put something\n" | timeout $TIMEOUT nc $HOST $PORT 2>/dev/null
echo "Testing: Invalid command 'put' -> $(printf "put something\n" | timeout $TIMEOUT nc $HOST $PORT 2>/dev/null || echo "ERROR: No response")"

printf "delete test\n" | timeout $TIMEOUT nc $HOST $PORT 2>/dev/null
echo "Testing: Invalid command 'delete' -> $(printf "delete test\n" | timeout $TIMEOUT nc $HOST $PORT 2>/dev/null || echo "ERROR: No response")"

printf "invalid command\n" | timeout $TIMEOUT nc $HOST $PORT 2>/dev/null
echo "Testing: Invalid command -> $(printf "invalid command\n" | timeout $TIMEOUT nc $HOST $PORT 2>/dev/null || echo "ERROR: No response")"
echo

echo "7. EMPTY OR SPECIAL CASES:"
test_address "SRS0=" "SRS0 empty"
test_address "SRS1=" "SRS1 empty"
echo

echo "=== Test completed ==="
echo "Note: Responses starting with '200' are successful, '500' indicate errors"
echo "If you see 'ERROR: No response', make sure srs-resolver is running on $HOST:$PORT"