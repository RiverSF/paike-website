-- 订单联系电话列改名：parent_phone → contact_phone。
-- 原因：该列在收入订单里存学员家长电话、在开支订单里存授课老师电话，用「家长」命名已不准确。
-- 业务代码会在启动时自动完成改名（AutoMigrate 之前执行，见 model.MigrateOrderContactPhone）；
-- 本文件供手工执行 / 存量库对账使用，可重复执行。

-- 情况一：旧列存在、新列不存在 → 直接改名（保留数据）
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.columns
              WHERE table_name = 'order' AND column_name = 'parent_phone')
     AND NOT EXISTS (SELECT 1 FROM information_schema.columns
              WHERE table_name = 'order' AND column_name = 'contact_phone') THEN
    ALTER TABLE "order" RENAME COLUMN parent_phone TO contact_phone;
  END IF;
END $$;

-- 情况二：新列已由 AutoMigrate 建出（空列）→ 先搬值，再删旧列，避免数据留在旧列上
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.columns
              WHERE table_name = 'order' AND column_name = 'parent_phone')
     AND EXISTS (SELECT 1 FROM information_schema.columns
              WHERE table_name = 'order' AND column_name = 'contact_phone') THEN
    UPDATE "order" SET contact_phone = parent_phone
      WHERE (contact_phone IS NULL OR contact_phone = '') AND parent_phone <> '';
    ALTER TABLE "order" DROP COLUMN parent_phone;
  END IF;
END $$;

COMMENT ON COLUMN "order".contact_phone IS '对方联系电话（收入=学员家长，开支=授课老师）';
ALTER TABLE "order" ALTER COLUMN contact_phone SET DEFAULT '';
ALTER TABLE "order" ALTER COLUMN contact_phone SET NOT NULL;
