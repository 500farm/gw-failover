package main

import "time"

const InactiveRouteMetric = 10000
const DownThreshold = 30 * time.Second
const UpThreshold = 120 * time.Second
const PingInterval = time.Second
const AcceptableReplyRatio = 0.9

const DryRun = true
