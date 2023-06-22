/**
 * Copyright 2023 The Sigstore Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Enable required services for this module
resource "google_compute_security_policy" "fulcio" {
  name    = "fulcio-service-security-policy"
  project = var.project_id
  type    = "CLOUD_ARMOR"

  rule {
    action   = "throttle"
    priority = "1"
    match {
      versioned_expr = "SRC_IPS_V1"
      config {
        src_ip_ranges = ["*"]
      }
    }
    rate_limit_options {
      enforce_on_key = "IP"
      conform_action = "allow"
      exceed_action  = "deny(429)"
      rate_limit_threshold {
        count        = "15"
        interval_sec = "60"
      }
    }
    description = "Rate limit all traffic by client IP"
  }

  rule {
    action   = "allow"
    priority = "2147483647"
    match {
      versioned_expr = "SRC_IPS_V1"
      config {
        src_ip_ranges = ["*"]
      }
    }
    description = "default rule"
  }

  advanced_options_config {
    json_parsing = "STANDARD"
  }

  adaptive_protection_config {
    layer_7_ddos_defense_config {
      enable = true
    }
  }
}

