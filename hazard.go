package conduit

import pgdiff "github.com/stripe/pg-schema-diff/pkg/diff"

// HazardType identifies the category of risk a migration statement carries.
//
// Hazardous operation types re-exported from pg-schema-diff.
// Pass these to MigrateOptions.AllowHazards to permit specific categories
// of risky operations.
type HazardType = pgdiff.MigrationHazardType

const (
	HazardTypeAcquiresAccessExclusiveLock   HazardType = pgdiff.MigrationHazardTypeAcquiresAccessExclusiveLock
	HazardTypeAcquiresShareLock             HazardType = pgdiff.MigrationHazardTypeAcquiresShareLock
	HazardTypeAcquiresShareRowExclusiveLock HazardType = pgdiff.MigrationHazardTypeAcquiresShareRowExclusiveLock
	HazardTypeCorrectness                   HazardType = pgdiff.MigrationHazardTypeCorrectness
	HazardTypeDeletesData                   HazardType = pgdiff.MigrationHazardTypeDeletesData
	HazardTypeHasUntrackableDependencies    HazardType = pgdiff.MigrationHazardTypeHasUntrackableDependencies
	HazardTypeIndexBuild                    HazardType = pgdiff.MigrationHazardTypeIndexBuild
	HazardTypeIndexDropped                  HazardType = pgdiff.MigrationHazardTypeIndexDropped
	HazardTypeImpactsDatabasePerformance    HazardType = pgdiff.MigrationHazardTypeImpactsDatabasePerformance
	HazardTypeIsUserGenerated               HazardType = pgdiff.MigrationHazardTypeIsUserGenerated
	HazardTypeExtensionVersionUpgrade       HazardType = pgdiff.MigrationHazardTypeExtensionVersionUpgrade
	HazardTypeAuthzUpdate                   HazardType = pgdiff.MigrationHazardTypeAuthzUpdate
)
