-- Remove campos securitários por nova versão apenas quando presentes; preserva modelos já adequados.
DO $migration$
DECLARE
    target_template_id uuid;
    target_creator_id uuid;
    old_version integer;
    original_markdown text;
    revised_markdown text;
    operation_start integer;
    insurance_start integer;
    pricing_start integer;
BEGIN
    SELECT t.id, t.created_by, t.current_version, v.markdown
    INTO target_template_id, target_creator_id, old_version, original_markdown
    FROM contract_templates t
    JOIN contract_template_versions v
      ON v.contract_template_id = t.id
     AND v.version_number = t.current_version
    WHERE t.is_active = true
      AND t.is_default = true
    FOR UPDATE OF t;

    IF NOT FOUND THEN
        IF EXISTS (SELECT 1 FROM contract_templates) THEN
            RAISE EXCEPTION 'active default contract template not found';
        END IF;
        RETURN;
    END IF;

    operation_start := strpos(original_markdown, '### 1.3. Operação');
    insurance_start := strpos(original_markdown, '### 1.4. Informações securitárias');
    pricing_start := strpos(original_markdown, '## 2. PREÇO E FORMA DE PAGAMENTO');

    -- Modelos já adequados são preservados, sem criar versões artificiais.
    IF operation_start = 0 AND insurance_start = 0 THEN
        IF pricing_start = 0
           OR strpos(lower(original_markdown), '{operation.') > 0
           OR strpos(lower(original_markdown), '{insurance.') > 0 THEN
            RAISE EXCEPTION 'default contract template has unresolved operational insurance fields';
        END IF;
        RETURN;
    END IF;

    IF operation_start = 0 OR insurance_start <= operation_start OR pricing_start <= insurance_start THEN
        RAISE EXCEPTION 'default contract template has unexpected operation or insurance sections';
    END IF;

    revised_markdown :=
        rtrim(left(original_markdown, operation_start - 1), E'\n')
        || E'\n\n'
        || substr(original_markdown, pricing_start);

    IF revised_markdown ~ '\{(operation|insurance)\.' THEN
        RAISE EXCEPTION 'contract template still depends on operational insurance variables';
    END IF;

    INSERT INTO contract_template_versions (
        contract_template_id,
        version_number,
        markdown,
        template_hash,
        created_by
    )
    VALUES (
        target_template_id,
        old_version + 1,
        revised_markdown,
        digest(revised_markdown, 'sha256'),
        target_creator_id
    );

    UPDATE contract_templates
    SET current_version = old_version + 1,
        updated_at = now()
    WHERE id = target_template_id;
END;
$migration$;
