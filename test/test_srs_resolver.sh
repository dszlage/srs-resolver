#!/bin/bash

# Test script for srs-resolver
# Usage: ./test_srs_resolver.sh
# Make sure srs-resolver is running on 127.0.0.1:12345

HOST="127.0.0.1"
PORT="12345"
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

echo "1. PRAWIDŁOWE ADRESY EMAIL (najczęstszy przypadek):"
test_address "damian@example.com" "Standardowy email"
test_address "user@driftzone24.pl" "Email z polską domeną"
test_address "test.email@subdomain.example.org" "Email z subdomeną"
test_address "admin+tag@company.com" "Email z tagiem"
echo

echo "2. PRAWIDŁOWE ADRESY SRS0:"
test_address "SRS0=abc123=timestamp=originaldomain.com=localpart" "SRS0 podstawowy"
test_address "SRS0=hash123=12345=example.com=damian" "SRS0 z timestampem"
test_address "SRS0=xyz789=67890=driftzone24.pl=admin" "SRS0 z polską domeną"
echo

echo "3. PRAWIDŁOWE ADRESY SRS1:"
test_address "SRS1=def456=timestamp=originaldomain.com=localpart" "SRS1 podstawowy"
test_address "SRS1=hash456=12345=example.com=user.name" "SRS1 z kropką w nazwie"
echo

echo "4. SRS Z PEŁNYM ADRESEM W CZĘŚCI LOKALNEJ:"
test_address "SRS0=abc123=12345=driftzone24.pl=damian.szlage@attmail.pl" "SRS0 z pełnym adresem"
test_address "SRS1=def456=67890=example.com=user@forwarder.com" "SRS1 z pełnym adresem"
echo

echo "5. BŁĘDNE FORMATY (testowanie fallback):"
test_address "invalid-email-without-at" "Email bez @"
test_address "email@" "Email z pustą domeną"
test_address "@domain.com" "Email z pustą częścią lokalną"
test_address "email with spaces@domain.com" "Email ze spacjami"
test_address "email<with>forbidden@chars.com" "Email z zabronionymi znakami"
test_address "SRS0=incomplete" "Niepełny SRS0"
test_address "SRS0=too=few=parts" "SRS0 z za małą liczbą części"
test_address "SRS0=too=many=equal=signs=here=and=more" "SRS0 z za dużą liczbą części"
test_address "completely-invalid-input" "Całkowicie nieprawidłowe dane"
echo

echo "6. NIEPRAWIDŁOWE KOMENDY PROTOKOŁU:"
test_address "" "Pusta komenda (tylko get)"
printf "put something\n" | timeout $TIMEOUT nc $HOST $PORT 2>/dev/null
echo "Testing: Nieprawidłowa komenda 'put' -> $(printf "put something\n" | timeout $TIMEOUT nc $HOST $PORT 2>/dev/null || echo "ERROR: No response")"

printf "delete test\n" | timeout $TIMEOUT nc $HOST $PORT 2>/dev/null
echo "Testing: Nieprawidłowa komenda 'delete' -> $(printf "delete test\n" | timeout $TIMEOUT nc $HOST $PORT 2>/dev/null || echo "ERROR: No response")"

printf "invalid command\n" | timeout $TIMEOUT nc $HOST $PORT 2>/dev/null
echo "Testing: Nieprawidłowa komenda -> $(printf "invalid command\n" | timeout $TIMEOUT nc $HOST $PORT 2>/dev/null || echo "ERROR: No response")"
echo

echo "7. PUSTE LUB SPECJALNE PRZYPADKI:"
test_address "SRS0=" "SRS0 pusty"
test_address "SRS1=" "SRS1 pusty"
echo

echo "=== Test completed ==="
echo "Note: Responses starting with '200' are successful, '500' indicate errors"
echo "If you see 'ERROR: No response', make sure srs-resolver is running on $HOST:$PORT"