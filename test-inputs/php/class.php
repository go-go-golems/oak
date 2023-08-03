<?php

declare(strict_types=1);

namespace MyApp;

use MyApp\Util\ExampleUtil;

/**
 * An example class to demonstrate syntax.
 */
class ExampleClass
{
    private readonly int $var1;

    public function __construct(
        int $var1
    ) {
        $this->var1 = $var1;
    }

    /**
     * An example function.
     *
     * @return int
     */
    public function exampleFunction(): int
    {
        return ExampleUtil::exampleFunction($this->var1);
    }
}
