#!/bin/bash

BOOTSTRAP_FORCE=false
BOOTSTRAP_ONLY_SKILLS=false
BOOTSTRAP_WORK_DIR=""
BOOTSTRAP_STAGE_DIR=""
BOOTSTRAP_BACKUP_DIR=""
BOOTSTRAP_SUCCESS=false
BOOTSTRAP_CREATED_README=false
BOOTSTRAP_REPLACED_AGENTS=false
BOOTSTRAP_REPLACED_SKILLS=false
BOOTSTRAP_RULES_WRITTEN=()
BOOTSTRAP_STALE_RULES_REMOVED=()

if ! command -v require_runtime_vars >/dev/null 2>&1; then
  require_runtime_vars() {
    local var_name
    local var_value

    for var_name in "$@"; do
      var_value="${!var_name-}"
      if [ -z "$var_value" ]; then
        echo "[ERR] Missing required runtime variable: $var_name" >&2
        return 1
      fi
    done
  }
fi

bootstrap_require_context() {
  require_runtime_vars AGENT47_HOME TEMPLATES_DIR RULES_TEMPLATES_DIR AGENT47_SKILL_UTILS PROJECT_RULES_DIR PROJECT_SKILLS_DIR PROJECT_AGENTS_FILE PROJECT_README_FILE
  assert_manifest_contract
}

bootstrap_cleanup_transaction() {
  if [ -n "$BOOTSTRAP_WORK_DIR" ] && [ -d "$BOOTSTRAP_WORK_DIR" ]; then
    rm -rf "$BOOTSTRAP_WORK_DIR"
  fi
}

bootstrap_rollback_transaction() {
  local file

  if [ "$BOOTSTRAP_SUCCESS" = true ]; then
    return 0
  fi

  if [ "$BOOTSTRAP_CREATED_README" = true ]; then
    rm -f "$PROJECT_README_FILE"
  fi

  if [ "$BOOTSTRAP_REPLACED_AGENTS" = true ]; then
    if [ -f "$BOOTSTRAP_BACKUP_DIR/$PROJECT_AGENTS_FILE" ]; then
      mv "$BOOTSTRAP_BACKUP_DIR/$PROJECT_AGENTS_FILE" "./$PROJECT_AGENTS_FILE"
    else
      rm -f "./$PROJECT_AGENTS_FILE"
    fi
  fi

  if [ "$BOOTSTRAP_REPLACED_SKILLS" = true ]; then
    rm -rf "$PROJECT_SKILLS_DIR"
    if [ -d "$BOOTSTRAP_BACKUP_DIR/$PROJECT_SKILLS_DIR" ]; then
      mv "$BOOTSTRAP_BACKUP_DIR/$PROJECT_SKILLS_DIR" "$PROJECT_SKILLS_DIR"
    fi
  fi

  for file in "${BOOTSTRAP_RULES_WRITTEN[@]}"; do
    if [ -f "$BOOTSTRAP_BACKUP_DIR/$PROJECT_RULES_DIR/$file" ]; then
      mv "$BOOTSTRAP_BACKUP_DIR/$PROJECT_RULES_DIR/$file" "$PROJECT_RULES_DIR/$file"
    else
      rm -f "$PROJECT_RULES_DIR/$file"
    fi
  done

  for file in "${BOOTSTRAP_STALE_RULES_REMOVED[@]}"; do
    if [ -f "$BOOTSTRAP_BACKUP_DIR/$PROJECT_RULES_DIR/$file" ]; then
      mv "$BOOTSTRAP_BACKUP_DIR/$PROJECT_RULES_DIR/$file" "$PROJECT_RULES_DIR/$file"
    fi
  done

  if [ -d "$PROJECT_RULES_DIR" ]; then
    rmdir "$PROJECT_RULES_DIR" >/dev/null 2>&1 || true
  fi
}

bootstrap_finish_transaction() {
  BOOTSTRAP_SUCCESS=true
  trap - EXIT INT TERM
  bootstrap_cleanup_transaction
}

bootstrap_prepare_transaction() {
  BOOTSTRAP_WORK_DIR="$(mktemp -d "${TMPDIR:-/tmp}/a47-stage-XXXXXX")"
  BOOTSTRAP_STAGE_DIR="$BOOTSTRAP_WORK_DIR/stage"
  BOOTSTRAP_BACKUP_DIR="$BOOTSTRAP_WORK_DIR/backup"
  mkdir -p "$BOOTSTRAP_STAGE_DIR" "$BOOTSTRAP_BACKUP_DIR"
  trap 'bootstrap_rollback_transaction; bootstrap_cleanup_transaction' EXIT INT TERM
}

bootstrap_setup_skills_stage() {
  local skills_dir="$1"
  local templates_dir="$AGENT47_HOME/templates/skills"
  local src dest_tmp dest_existing skill
  local skill_list=()

  echo "[INFO] Initializing agent skills..."
  mkdir -p "$skills_dir"

  while IFS= read -r skill; do
    [ -n "$skill" ] || continue
    skill_list+=("$skill")
  done < <(
    find "$templates_dir" -mindepth 2 -maxdepth 2 -type f -name 'SKILL.md' -print |
      while IFS= read -r path; do
        path="${path%/SKILL.md}"
        printf '%s\n' "${path##*/}"
      done | sort
  )

  if [ "$BOOTSTRAP_FORCE" != "true" ] && [ -d "$PROJECT_SKILLS_DIR" ]; then
    cp -R "$PROJECT_SKILLS_DIR/." "$skills_dir/"
  fi

  if [ "${#skill_list[@]}" -eq 0 ]; then
    echo "[ERR] No valid skill templates found in $templates_dir"
    return 1
  fi

  for skill in "${skill_list[@]}"; do
    src="$templates_dir/$skill"
    dest_tmp="$skills_dir/$skill"
    dest_existing="$PROJECT_SKILLS_DIR/$skill"

    if [ ! -d "$src" ]; then
      echo "[WARN] Template not found: $skill (skipping)"
      continue
    fi

    rm -rf "$dest_tmp"
    mkdir -p "$dest_tmp"

    if [ -d "$dest_existing" ] && [ "$BOOTSTRAP_FORCE" != "true" ]; then
      cp -R "$dest_existing/." "$dest_tmp/"
    else
      cp -R "$src/." "$dest_tmp/"
    fi

    if [ ! -f "$dest_tmp/SKILL.md" ]; then
      cp "$src/SKILL.md" "$dest_tmp/SKILL.md"
    fi

    if ! validate_skill "$dest_tmp/SKILL.md"; then
      if [ "$BOOTSTRAP_FORCE" = "true" ]; then
        echo "[ERR] Invalid skill template: $skill"
        return 1
      fi

      echo "[WARN] Invalid SKILL.md for $skill; preserving existing content"
    fi
  done

  write_available_skills "$skills_dir" "$skills_dir/AVAILABLE_SKILLS.xml"
  echo "[OK] Skills setup complete."
}

bootstrap_stage_project_files() {
  local file

  bootstrap_require_context || return 1

  bootstrap_setup_skills_stage "$BOOTSTRAP_STAGE_DIR/$PROJECT_SKILLS_DIR"

  if [ "$BOOTSTRAP_ONLY_SKILLS" = "true" ]; then
    return 0
  fi

  mkdir -p "$BOOTSTRAP_STAGE_DIR/$PROJECT_RULES_DIR"
  while IFS= read -r file; do
    cp "$RULES_TEMPLATES_DIR/$file" "$BOOTSTRAP_STAGE_DIR/$PROJECT_RULES_DIR/$file"
  done < <(project_rule_template_files)

  # TEMPLATES_DIR is injected by the calling command environment.
  # shellcheck disable=SC2153
  cp "$TEMPLATES_DIR/$PROJECT_AGENTS_FILE" "$BOOTSTRAP_STAGE_DIR/$PROJECT_AGENTS_FILE"
}

bootstrap_commit_skills() {
  if [ -d "$PROJECT_SKILLS_DIR" ]; then
    mv "$PROJECT_SKILLS_DIR" "$BOOTSTRAP_BACKUP_DIR/$PROJECT_SKILLS_DIR"
  fi
  mv "$BOOTSTRAP_STAGE_DIR/$PROJECT_SKILLS_DIR" "$PROJECT_SKILLS_DIR"
  BOOTSTRAP_REPLACED_SKILLS=true
}

bootstrap_commit_rules() {
  local file existing_rule target_path

  if [ ! -d "$PROJECT_RULES_DIR" ]; then
    mkdir "$PROJECT_RULES_DIR"
    echo "[OK] Created directory: $PROJECT_RULES_DIR/"
  fi

  mkdir -p "$BOOTSTRAP_BACKUP_DIR/$PROJECT_RULES_DIR"
  if [ "$BOOTSTRAP_FORCE" = true ]; then
    for existing_rule in "$PROJECT_RULES_DIR"/*.yaml; do
      [ -e "$existing_rule" ] || continue
      file="$(basename "$existing_rule")"
      if [ ! -f "$BOOTSTRAP_BACKUP_DIR/$PROJECT_RULES_DIR/$file" ]; then
        cp "$existing_rule" "$BOOTSTRAP_BACKUP_DIR/$PROJECT_RULES_DIR/$file"
      fi

      if ! manifest_contains_entry rule_templates "$file"; then
        rm -f "$existing_rule"
        BOOTSTRAP_STALE_RULES_REMOVED+=("$file")
        echo "[OK] Removed stale managed rule: $existing_rule"
      fi
    done
  fi

  while IFS= read -r file; do
    target_path="$PROJECT_RULES_DIR/$file"
    if [ -f "$target_path" ]; then
      if [ "$BOOTSTRAP_FORCE" = true ]; then
        if [ ! -f "$BOOTSTRAP_BACKUP_DIR/$PROJECT_RULES_DIR/$file" ]; then
          cp "$target_path" "$BOOTSTRAP_BACKUP_DIR/$PROJECT_RULES_DIR/$file"
        fi
        mv "$BOOTSTRAP_STAGE_DIR/$PROJECT_RULES_DIR/$file" "$target_path"
        BOOTSTRAP_RULES_WRITTEN+=("$file")
        echo "[OK] Updated: $target_path"
      else
        rm -f "$BOOTSTRAP_STAGE_DIR/$PROJECT_RULES_DIR/$file"
        echo "[WARN] $target_path already exists, skipping"
      fi
    else
      mv "$BOOTSTRAP_STAGE_DIR/$PROJECT_RULES_DIR/$file" "$target_path"
      BOOTSTRAP_RULES_WRITTEN+=("$file")
      echo "[OK] Copied: $target_path"
    fi
  done < <(project_rule_template_files)
}

bootstrap_commit_agents_file() {
  if [ -f "./$PROJECT_AGENTS_FILE" ]; then
    if [ "$BOOTSTRAP_FORCE" = true ]; then
      mv "./$PROJECT_AGENTS_FILE" "$BOOTSTRAP_BACKUP_DIR/$PROJECT_AGENTS_FILE"
      mv "$BOOTSTRAP_STAGE_DIR/$PROJECT_AGENTS_FILE" "./$PROJECT_AGENTS_FILE"
      BOOTSTRAP_REPLACED_AGENTS=true
      echo "[OK] Updated: $PROJECT_AGENTS_FILE"
    else
      rm -f "$BOOTSTRAP_STAGE_DIR/$PROJECT_AGENTS_FILE"
      echo "[WARN] $PROJECT_AGENTS_FILE already exists, skipping"
    fi
  else
    mv "$BOOTSTRAP_STAGE_DIR/$PROJECT_AGENTS_FILE" "./$PROJECT_AGENTS_FILE"
    BOOTSTRAP_REPLACED_AGENTS=true
    echo "[OK] Copied: $PROJECT_AGENTS_FILE"
  fi
}

bootstrap_commit_readme() {
  if [ ! -f "$PROJECT_README_FILE" ]; then
    touch "$PROJECT_README_FILE"
    BOOTSTRAP_CREATED_README=true
    echo "[OK] $PROJECT_README_FILE created"
  fi
}

bootstrap_require_templates() {
  local missing=0
  local file

  bootstrap_require_context || return 1

  if [ "$BOOTSTRAP_ONLY_SKILLS" != "true" ] && [ ! -f "$TEMPLATES_DIR/$PROJECT_AGENTS_FILE" ]; then
    echo "[ERR] Template not found: $PROJECT_AGENTS_FILE"
    missing=1
  fi

  if [ "$BOOTSTRAP_ONLY_SKILLS" != "true" ]; then
    while IFS= read -r file; do
      if [ ! -f "$RULES_TEMPLATES_DIR/$file" ]; then
        echo "[ERR] Template not found: $file"
        missing=1
      fi
    done < <(project_rule_template_files)
  fi

  if [ "$missing" -ne 0 ]; then
    echo "[ERR] Aborting: required templates missing."
    return 1
  fi
}

bootstrap_require_skills_support() {
  bootstrap_require_context || return 1

  if [ ! -f "$AGENT47_SKILL_UTILS" ]; then
    echo "[ERR] Aborting: missing helper dependency: $AGENT47_SKILL_UTILS"
    return 1
  fi

  if [ ! -d "$AGENT47_HOME/templates/skills" ]; then
    echo "[ERR] Aborting: missing skills templates: $AGENT47_HOME/templates/skills"
    return 1
  fi
}

bootstrap_usage() {
  echo "Usage: add-agent [--force] [--only-skills]"
}
