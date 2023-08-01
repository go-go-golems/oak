<?php

declare(strict_types=1);

namespace MyApp;

/**
 * This class demonstrates multiple types of method definitions.
 */
final class ExampleClass {
    /**
     * Public class method with no parameters and no return type.
     */
    public function publicNoParamNoReturn() { }

    /**
     * Private class method with parameters and no return type.
     */
    private function privateWithParamsNoReturn($param1, $param2) { }

    /**
     * Protected class method with default parameters and no return type.
     */
    protected function protectedWithDefaultParamsNoReturn($param1 = 1, $param2 = 'default') { }

    /**
     * Public static function with typed parameters and no return type.
     */
    public static function publicStaticWithTypedParamsNoReturn(int $param1, string $param2) { }

    /**
     * Private static method with no parameters and with a return type.
     */
    private static function privateStaticNoParamWithReturn(): int { return 1; }

    /**
     * Protected static method with parameters and a return type.
     */
    protected static function protectedStaticWithParamsAndReturn(int $param1, string $param2): array { return [$param1, $param2]; }

    /**
     * Public final function with default parameters and a return type.
     */
    public final function publicFinalWithDefaultParamsAndReturn($param1 = 1, $param2 = 'default'): bool { return true; }

    /**
     * Private function with nullable parameter type.
     */
    private function privateWithNullableParam(?int $param) { }
}

/**
 * Function with no parameters and no return type.
 */
function exampleFunc() { }

/**
 * Function with parameters and no return type.
 */
function exampleWithParamsNoReturn($param1, $param2) { }

/**
 * Function with default parameters and no return type.
 */
function exampleWithDefaultParamsNoReturn($param1 = 1, $param2 = 'default') { }

/**
 * Function with typed parameters and no return type.
 */
function exampleWithTypedParamsNoReturn(int $param1, string $param2) { }

/**
 * Function with no parameters and with a return type.
 */
function exampleNoParamWithReturn(): int { return 1; }

/**
 * Function with parameters and a return type.
 */
function exampleWithParamsAndReturn(int $param1, string $param2): array { return [$param1, $param2]; }

/**
 * Function with default parameters and a return type.
 */
function exampleWithDefaultParamsAndReturn($param1 = 1, $param2 = 'default'): bool { return true; }

/**
 * Function with nullable parameter type.
 */
function exampleFuncWithNullableParam(?int $param) { }
